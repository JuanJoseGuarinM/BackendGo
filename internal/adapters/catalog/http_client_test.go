package catalog

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"orders-api/internal/application"
	"orders-api/internal/platform/requestid"
)

// Verificación en compilación: los dos adaptadores cumplen el puerto.
var (
	_ application.CatalogClient = (*HTTPClient)(nil)
	_ application.CatalogClient = (*Static)(nil)
)

// fakeCatalog levanta un servidor HTTP REAL de prueba (httptest.Server) en un
// puerto local. handler decide qué responder; hits cuenta las llamadas.
func fakeCatalog(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func newClient(baseURL string) *HTTPClient {
	return NewHTTPClient(Config{
		BaseURL:        baseURL,
		AttemptTimeout: 100 * time.Millisecond,
		MaxRetries:     2,
		BaseBackoff:    time.Millisecond, // pruebas rápidas
	})
}

func okProduct(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"sku":"SKU-1","name":"Café","price_cents":1500}`))
}

func TestGetProduct_Success(t *testing.T) {
	srv, hits := fakeCatalog(t, okProduct)

	p, err := newClient(srv.URL).GetProduct(t.Context(), "SKU-1")

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if p.PriceCents != 1500 || p.Name != "Café" {
		t.Errorf("producto = %+v", p)
	}
	if hits.Load() != 1 {
		t.Errorf("llamadas = %d, se esperaba 1", hits.Load())
	}
}

func TestGetProduct_PropagatesRequestID(t *testing.T) {
	var got string
	srv, _ := fakeCatalog(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get(requestid.Header)
		okProduct(w, r)
	})

	ctx := requestid.With(t.Context(), "req-77")
	if _, err := newClient(srv.URL).GetProduct(ctx, "SKU-1"); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if got != "req-77" {
		t.Errorf("el catálogo recibió X-Request-ID = %q, se esperaba req-77", got)
	}
}

func TestGetProduct_NotFoundIsNotRetried(t *testing.T) {
	srv, hits := fakeCatalog(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := newClient(srv.URL).GetProduct(t.Context(), "NOPE")

	if !errors.Is(err, application.ErrProductNotFound) {
		t.Fatalf("error = %v, se esperaba ErrProductNotFound", err)
	}
	// 404 es una respuesta definitiva: reintentar no la cambia.
	if hits.Load() != 1 {
		t.Errorf("llamadas = %d, un 404 no se debe reintentar", hits.Load())
	}
}

func TestGetProduct_RetriesTemporaryFailure(t *testing.T) {
	var calls atomic.Int32
	srv, _ := fakeCatalog(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) <= 2 { // las 2 primeras fallan
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		okProduct(w, r)
	})

	p, err := newClient(srv.URL).GetProduct(t.Context(), "SKU-1")

	if err != nil {
		t.Fatalf("debía recuperarse con reintentos: %v", err)
	}
	if p.SKU != "SKU-1" || calls.Load() != 3 {
		t.Errorf("producto = %+v, llamadas = %d", p, calls.Load())
	}
}

// Evidencia del PDF: catálogo caído → error controlado.
func TestGetProduct_CatalogDown(t *testing.T) {
	srv, hits := fakeCatalog(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := newClient(srv.URL).GetProduct(t.Context(), "SKU-1")

	if !errors.Is(err, application.ErrCatalogUnavailable) {
		t.Fatalf("error = %v, se esperaba ErrCatalogUnavailable", err)
	}
	if hits.Load() != 3 { // 1 intento + 2 reintentos
		t.Errorf("llamadas = %d, se esperaban 3", hits.Load())
	}
}

func TestGetProduct_ServerNotRunning(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // apagado: conexión rechazada

	_, err := newClient(url).GetProduct(t.Context(), "SKU-1")

	if !errors.Is(err, application.ErrCatalogUnavailable) {
		t.Fatalf("error = %v, se esperaba ErrCatalogUnavailable", err)
	}
}

// El PDF pide probar el timeout con httptest.Server.
func TestGetProduct_Timeout(t *testing.T) {
	srv, _ := fakeCatalog(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second): // el catálogo está MUY lento
			okProduct(w, r)
		case <-r.Context().Done(): // el cliente se rindió
		}
	})

	start := time.Now()
	_, err := newClient(srv.URL).GetProduct(t.Context(), "SKU-1")
	elapsed := time.Since(start)

	if !errors.Is(err, application.ErrCatalogUnavailable) {
		t.Fatalf("error = %v, se esperaba ErrCatalogUnavailable", err)
	}
	// 3 intentos de 100ms + esperas cortas. Sin timeout tardaría 6 segundos.
	if elapsed > time.Second {
		t.Errorf("tardó %v: el timeout por intento no funcionó", elapsed)
	}
}

func TestGetProduct_StopsRetryingWhenCallerCancels(t *testing.T) {
	srv, hits := fakeCatalog(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	client := NewHTTPClient(Config{BaseURL: srv.URL, MaxRetries: 5, BaseBackoff: 200 * time.Millisecond})

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	_, err := client.GetProduct(ctx, "SKU-1")

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, se esperaba context.DeadlineExceeded", err)
	}
	// Si quien llama ya no espera, no tiene sentido seguir reintentando.
	if hits.Load() != 1 {
		t.Errorf("llamadas = %d, debía parar tras la cancelación", hits.Load())
	}
}
