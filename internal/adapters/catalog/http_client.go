// Package catalog tiene los ADAPTADORES hacia el catálogo de productos.
//
// HTTPClient habla con un catálogo real por HTTP. Su trabajo es aislar a la
// aplicación de todo lo que puede salir mal en la red: lentitud, caídas,
// respuestas raras. Afuera solo salen dos errores conocidos:
// application.ErrProductNotFound y application.ErrCatalogUnavailable.
package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"time"

	"orders-api/internal/application"
	"orders-api/internal/platform/requestid"
)

// Config de la integración. Cada valor es una decisión de resiliencia.
type Config struct {
	BaseURL        string
	AttemptTimeout time.Duration // máximo por intento: no esperar para siempre a un servicio lento
	MaxRetries     int           // reintentos ADEMÁS del primer intento
	BaseBackoff    time.Duration // espera antes del 1er reintento; se duplica en cada uno
}

// HTTPClient implementa application.CatalogClient por HTTP.
type HTTPClient struct {
	cfg  Config
	http *http.Client
}

func NewHTTPClient(cfg Config) *HTTPClient {
	if cfg.AttemptTimeout <= 0 {
		cfg.AttemptTimeout = 2 * time.Second
	}
	if cfg.BaseBackoff <= 0 {
		cfg.BaseBackoff = 100 * time.Millisecond
	}
	// Un http.Client propio y reutilizable (reutiliza conexiones). Nunca
	// http.DefaultClient: no tiene timeout. Aquí el límite lo pone el context
	// de cada intento.
	return &HTTPClient{cfg: cfg, http: &http.Client{}}
}

// errRetryable marca fallas TEMPORALES que vale la pena reintentar. Es
// privado: nunca sale de este paquete.
var errRetryable = errors.New("falla temporal del catálogo")

// GetProduct consulta un producto, con reintentos y backoff exponencial.
//
// ¿Por qué se puede reintentar? Porque es un GET: es SEGURO (no cambia nada)
// e IDEMPOTENTE (pedirlo 1 o 3 veces da lo mismo). Reintentar un POST que
// crea o cobra algo podría duplicarlo.
func (c *HTTPClient) GetProduct(ctx context.Context, sku string) (application.Product, error) {
	var lastErr error

	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			wait := c.backoff(attempt)
			// Observabilidad (J6): sin este log, los reintentos serían invisibles
			// y solo veríamos "la API está lenta" sin saber por qué.
			slog.WarnContext(ctx, "reintentando catálogo",
				"sku", sku, "attempt", attempt, "wait_ms", wait.Milliseconds(), "error", lastErr)

			// Esperar antes de reintentar, pero respetando la cancelación.
			if err := sleep(ctx, wait); err != nil {
				return application.Product{}, fmt.Errorf("%w: %w", application.ErrCatalogUnavailable, err)
			}
		}

		p, err := c.getOnce(ctx, sku)
		if err == nil {
			return p, nil
		}
		if !errors.Is(err, errRetryable) {
			return application.Product{}, err // 404, respuesta inválida...: reintentar no ayuda
		}
		lastErr = err
	}

	return application.Product{}, fmt.Errorf("%w: falló %d veces: %v",
		application.ErrCatalogUnavailable, c.cfg.MaxRetries+1, lastErr)
}

// productDTO es el contrato JSON del catálogo (su formato, no el nuestro).
type productDTO struct {
	SKU        string `json:"sku"`
	Name       string `json:"name"`
	PriceCents int64  `json:"price_cents"`
}

// getOnce hace UN intento y clasifica el resultado.
func (c *HTTPClient) getOnce(ctx context.Context, sku string) (application.Product, error) {
	// Timeout por intento, encima del ctx de quien llama: gana el que venza primero.
	ctx, cancel := context.WithTimeout(ctx, c.cfg.AttemptTimeout)
	defer cancel()

	u := c.cfg.BaseURL + "/products/" + url.PathEscape(sku) // PathEscape: un SKU raro no rompe la URL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return application.Product{}, fmt.Errorf("armar solicitud al catálogo: %w", err)
	}
	// Propagamos el correlation ID al otro servicio: así una misma solicitud se
	// puede seguir en los logs de la API Y del catálogo (base del tracing).
	if id := requestid.From(ctx); id != "" {
		req.Header.Set(requestid.Header, id)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		// Red caída, conexión rechazada o venció el timeout del intento.
		return application.Product{}, fmt.Errorf("%w: %v", errRetryable, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		var dto productDTO
		// LimitReader: no confiar en que el otro servicio mande un body razonable.
		if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&dto); err != nil {
			return application.Product{}, fmt.Errorf("%w: respuesta inválida: %v", application.ErrCatalogUnavailable, err)
		}
		return application.Product{SKU: dto.SKU, Name: dto.Name, PriceCents: dto.PriceCents}, nil

	case resp.StatusCode == http.StatusNotFound:
		return application.Product{}, fmt.Errorf("sku %q: %w", sku, application.ErrProductNotFound)

	case resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests:
		// 5xx = el catálogo tiene un problema temporal; 429 = nos pide ir más lento.
		return application.Product{}, fmt.Errorf("%w: respondió %d", errRetryable, resp.StatusCode)

	default:
		return application.Product{}, fmt.Errorf("%w: respuesta inesperada %d", application.ErrCatalogUnavailable, resp.StatusCode)
	}
}

// backoff: 100ms, 200ms, 400ms... + un poco de azar (jitter). El jitter evita
// que muchos clientes reintenten exactamente al mismo tiempo y vuelvan a
// tumbar al servicio que se está recuperando.
func (c *HTTPClient) backoff(attempt int) time.Duration {
	d := c.cfg.BaseBackoff << (attempt - 1)
	// math/rand (no crypto/rand) a propósito: el jitter no es un dato de
	// seguridad, solo necesita variar. Un linter de seguridad (gosec) lo marca
	// igual, por eso se le indica explícitamente que es intencional.
	return d + rand.N(d/2+1) //nolint:gosec // jitter, no requiere aleatoriedad criptográfica
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
