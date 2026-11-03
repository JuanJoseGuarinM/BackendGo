// Punto de entrada de la API. main solo ARMA las piezas y arranca:
// no tiene lógica de negocio.
package main

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"orders-api/internal/adapters/catalog"
	"orders-api/internal/adapters/classifier"
	"orders-api/internal/adapters/httpapi"
	"orders-api/internal/adapters/repository/memory"
	"orders-api/internal/application"
	"orders-api/internal/platform/logging"
)

func main() {
	// Logs estructurados (J6): JSON por defecto, "text" para leer en local.
	// "service" va en cada línea para distinguir API y catálogo en un mismo visor.
	slog.SetDefault(logging.New(os.Stdout, os.Getenv("LOG_FORMAT"), os.Getenv("LOG_LEVEL")).With("service", "orders-api"))

	if err := run(); err != nil {
		slog.Error("la API terminó con error", "error", err)
		os.Exit(1)
	}
}

// run separado de main: devuelve error en vez de llamar os.Exit en varios
// lugares, y los defer alcanzan a ejecutarse.
func run() error {
	// 1. Armar dependencias (de adentro hacia afuera).
	repo := memory.NewOrderRepository()
	svc := application.NewOrderService(repo, newCatalog(), rand.Text)

	// Clasificador IA simulado (J7) con reglas fijas de respaldo.
	// CLASSIFIER_MODE: "" (ok), "fail", "invalid" o "slow", para ver cada riesgo.
	model := classifier.Simulated{Mode: os.Getenv("CLASSIFIER_MODE"), Latency: 150 * time.Millisecond}
	classification := application.NewClassificationService(repo, model, classifier.Rules{}, 1*time.Second)

	gin.SetMode(gin.ReleaseMode)
	ready := &httpapi.Readiness{} // empieza en "no listo"
	router := httpapi.NewRouter(httpapi.Deps{Orders: svc, Classifier: classification, Ready: ready})

	// 2. Configurar el servidor. Configuración por variable de entorno.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second, // protege contra clientes lentos a propósito
	}

	// 3. Apagado controlado (graceful shutdown): al recibir Ctrl+C o SIGTERM
	// (lo que envía Kubernetes), dejamos de aceptar solicitudes y terminamos
	// las que están en curso antes de salir.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("API escuchando", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()
	ready.SetReady(true) // todo armado: /readyz empieza a responder 200

	select {
	case err := <-errCh:
		return err // no pudo arrancar (ej. puerto ocupado)
	case <-ctx.Done():
	}

	// 4. Drenado: primero decimos "ya no estoy listo" (/readyz → 503) y
	// esperamos un poco para que Kubernetes deje de enviarnos tráfico nuevo.
	// Recién ahí cerramos. Sin esto, durante un despliegue algunas solicitudes
	// llegarían a un pod que ya se está apagando.
	ready.SetReady(false)
	delay, _ := time.ParseDuration(os.Getenv("SHUTDOWN_DELAY")) // ej. "5s" en Kubernetes; vacío = 0
	slog.Info("apagando la API...", "drain_delay", delay)
	time.Sleep(delay)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// newCatalog elige el adaptador del catálogo según la configuración.
// Sin CATALOG_URL se usa el catálogo en memoria, para poder correr la API sola.
func newCatalog() application.CatalogClient {
	baseURL := os.Getenv("CATALOG_URL")
	if baseURL == "" {
		slog.Warn("CATALOG_URL vacío: usando catálogo en memoria")
		return catalog.NewStatic(catalog.SampleProducts()...)
	}
	slog.Info("usando catálogo HTTP", "url", baseURL)
	return catalog.NewHTTPClient(catalog.Config{
		BaseURL:        baseURL,
		AttemptTimeout: 1 * time.Second,
		MaxRetries:     2,
		BaseBackoff:    100 * time.Millisecond,
	})
}
