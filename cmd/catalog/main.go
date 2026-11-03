// Catálogo FICTICIO para practicar la integración. Es otro servicio, con su
// propio proceso y puerto. Solo usa la librería estándar (net/http).
//
//	go run ./cmd/catalog                      → puerto 8081
//	$env:CATALOG_DELAY="3s"; go run ./cmd/catalog  → responde lento (para ver el timeout)
package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"orders-api/internal/adapters/catalog"
	"orders-api/internal/platform/logging"
	"orders-api/internal/platform/requestid"
)

type productJSON struct {
	SKU        string `json:"sku"`
	Name       string `json:"name"`
	PriceCents int64  `json:"price_cents"`
}

func main() {
	slog.SetDefault(logging.New(os.Stdout, os.Getenv("LOG_FORMAT"), os.Getenv("LOG_LEVEL")).With("service", "catalog"))

	products := map[string]productJSON{}
	for _, p := range catalog.SampleProducts() {
		products[p.SKU] = productJSON{SKU: p.SKU, Name: p.Name, PriceCents: p.PriceCents}
	}

	delay, _ := time.ParseDuration(os.Getenv("CATALOG_DELAY"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /products/{sku}", func(w http.ResponseWriter, r *http.Request) {
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-r.Context().Done(): // el cliente se rindió (timeout): dejamos de esperar
				return
			}
		}
		p, ok := products[r.PathValue("sku")]
		// El X-Request-ID que manda la API llega hasta aquí: con el mismo ID
		// se ve la solicitud en los logs de los DOS servicios.
		ctx := requestid.With(r.Context(), r.Header.Get(requestid.Header))
		slog.InfoContext(ctx, "consulta de producto", "sku", r.PathValue("sku"), "found", ok)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(p)
	})
	// Health check para Kubernetes. Este servicio no tiene dependencias ni
	// estado que preparar, así que el mismo endpoint sirve para liveness y readiness.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	slog.Info("catálogo escuchando", "addr", ":"+port, "delay", delay)
	srv := &http.Server{Addr: ":" + port, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("catálogo terminó", "error", err)
		os.Exit(1)
	}
}
