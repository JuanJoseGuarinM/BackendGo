package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"orders-api/internal/adapters/catalog"
	"orders-api/internal/adapters/classifier"
	"orders-api/internal/adapters/repository/memory"
	"orders-api/internal/application"
	"orders-api/internal/domain/order"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode) // silencia los logs de Gin en las pruebas
	m.Run()
}

// newTestRouter arma la API real (servicio + repositorio y catálogo en
// memoria) con IDs predecibles: o-1, o-2, ...
func newTestRouter() *gin.Engine {
	return newTestRouterWithCatalog(catalog.NewStatic(catalog.SampleProducts()...))
}

func newTestRouterWithCatalog(cat application.CatalogClient) *gin.Engine {
	return newTestRouterWith(cat, classifier.Simulated{})
}

// newTestRouterWith arma la API completa; los dos servicios comparten el
// mismo repositorio, como en cmd/api.
func newTestRouterWith(cat application.CatalogClient, model application.OrderClassifier) *gin.Engine {
	n := 0
	newID := func() string { n++; return fmt.Sprintf("o-%d", n) }
	repo := memory.NewOrderRepository()
	return NewRouter(Deps{
		Orders:     application.NewOrderService(repo, cat, newID),
		Classifier: application.NewClassificationService(repo, model, classifier.Rules{}, 100*time.Millisecond),
		Ready:      &Readiness{},
	})
}

// downCatalog simula el catálogo caído (lo que devolvería HTTPClient tras
// agotar los reintentos).
type downCatalog struct{}

func (downCatalog) GetProduct(context.Context, string) (application.Product, error) {
	return application.Product{}, fmt.Errorf("%w: timeout", application.ErrCatalogUnavailable)
}

func TestCreateOrder_CatalogErrors(t *testing.T) {
	t.Run("producto no existe en el catálogo", func(t *testing.T) {
		rec := do(newTestRouter(), http.MethodPost, "/orders", `{"customer_id":"c1","items":[{"sku":"NOPE","quantity":1}]}`)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d\n%s", rec.Code, rec.Body.String())
		}
		if got := decode[errorResponse](t, rec); got.Code != "PRODUCT_NOT_FOUND" {
			t.Errorf("code = %q", got.Code)
		}
	})

	// Evidencia del PDF: catálogo caído → error controlado (503), no un 500.
	t.Run("catálogo caído", func(t *testing.T) {
		rec := do(newTestRouterWithCatalog(downCatalog{}), http.MethodPost, "/orders", validOrder)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d\n%s", rec.Code, rec.Body.String())
		}
		if got := decode[errorResponse](t, rec); got.Code != "CATALOG_UNAVAILABLE" {
			t.Errorf("code = %q", got.Code)
		}
	})

	t.Run("el precio lo pone el catálogo, no el cliente", func(t *testing.T) {
		body := `{"customer_id":"c1","items":[{"sku":"SKU-1","quantity":1,"unit_price":0}]}`
		rec := do(newTestRouter(), http.MethodPost, "/orders", body)

		if got := decode[orderResponse](t, rec); got.Total != 1500 {
			t.Errorf("total = %d, se esperaba 1500 (precio del catálogo)", got.Total)
		}
	})
}

// do ejecuta una solicitud contra el router SIN levantar un servidor real.
// httptest.NewRecorder captura la respuesta para revisarla.
func do(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v\n%s", err, rec.Body.String())
	}
	return v
}

const validOrder = `{"customer_id":"c1","items":[{"sku":"SKU-1","quantity":2}]}`

func TestCreateOrder(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string // solo para errores
	}{
		{name: "éxito", body: validOrder, wantStatus: http.StatusCreated},
		{name: "JSON mal formado", body: `{"customer_id":`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
		{name: "falta customer_id", body: `{"items":[]}`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
		{name: "cantidad cero", body: `{"customer_id":"c1","items":[{"sku":"A","quantity":0}]}`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(newTestRouter(), http.MethodPost, "/orders", tt.body)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, se esperaba %d\n%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantCode != "" {
				if got := decode[errorResponse](t, rec); got.Code != tt.wantCode {
					t.Errorf("code = %q, se esperaba %q", got.Code, tt.wantCode)
				}
			}
		})
	}
}

func TestCreateOrder_ResponseBody(t *testing.T) {
	rec := do(newTestRouter(), http.MethodPost, "/orders", validOrder)

	got := decode[orderResponse](t, rec)
	if got.ID != "o-1" || got.Status != "PENDING" || got.Total != 3000 {
		t.Errorf("respuesta = %+v", got)
	}
	if loc := rec.Header().Get("Location"); loc != "/orders/o-1" {
		t.Errorf("Location = %q", loc)
	}
}

func TestGetOrder(t *testing.T) {
	r := newTestRouter()
	do(r, http.MethodPost, "/orders", validOrder) // crea o-1

	t.Run("existe", func(t *testing.T) {
		rec := do(r, http.MethodGet, "/orders/o-1", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("no existe", func(t *testing.T) {
		rec := do(r, http.MethodGet, "/orders/no-existe", "")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d", rec.Code)
		}
		if got := decode[errorResponse](t, rec); got.Code != "ORDER_NOT_FOUND" {
			t.Errorf("code = %q", got.Code)
		}
	})
}

func TestConfirmOrder(t *testing.T) {
	r := newTestRouter()
	do(r, http.MethodPost, "/orders", validOrder)                        // o-1: con productos
	do(r, http.MethodPost, "/orders", `{"customer_id":"c1","items":[]}`) // o-2: vacío

	tests := []struct {
		name       string
		id         string
		wantStatus int
		wantCode   string
	}{
		// El orden importa: el tercer caso depende de que el primero ya confirmó o-1.
		{name: "confirma pedido con productos", id: "o-1", wantStatus: http.StatusOK},
		{name: "pedido vacío (regla de negocio)", id: "o-2", wantStatus: http.StatusUnprocessableEntity, wantCode: "ORDER_EMPTY"},
		{name: "ya confirmado (conflicto de estado)", id: "o-1", wantStatus: http.StatusConflict, wantCode: "ORDER_ALREADY_CONFIRMED"},
		{name: "no existe", id: "o-99", wantStatus: http.StatusNotFound, wantCode: "ORDER_NOT_FOUND"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(r, http.MethodPost, "/orders/"+tt.id+"/confirm", "")

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, se esperaba %d\n%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantCode != "" {
				if got := decode[errorResponse](t, rec); got.Code != tt.wantCode {
					t.Errorf("code = %q, se esperaba %q", got.Code, tt.wantCode)
				}
			}
		})
	}
}

// brokenService simula una falla inesperada (ej. la base de datos caída).
type brokenService struct{}

func (brokenService) Create(context.Context, application.CreateOrderInput) (*order.Order, error) {
	return nil, errors.New("conexión a la base de datos perdida: host=10.0.0.5")
}
func (brokenService) Get(context.Context, string) (*order.Order, error)     { return nil, nil }
func (brokenService) Confirm(context.Context, string) (*order.Order, error) { return nil, nil }

func TestUnexpectedError_DoesNotLeakDetails(t *testing.T) {
	rec := do(NewRouter(Deps{Orders: brokenService{}, Ready: &Readiness{}}), http.MethodPost, "/orders", validOrder)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "10.0.0.5") {
		t.Errorf("la respuesta expone detalles internos: %s", rec.Body.String())
	}
}

func TestRequestID(t *testing.T) {
	r := newTestRouter()

	t.Run("genera uno si no viene", func(t *testing.T) {
		rec := do(r, http.MethodGet, "/orders/x", "")
		id := rec.Header().Get(HeaderRequestID)
		if id == "" {
			t.Fatal("falta el header X-Request-ID")
		}
		if got := decode[errorResponse](t, rec); got.CorrelationID != id {
			t.Errorf("correlation_id = %q, header = %q", got.CorrelationID, id)
		}
	})

	t.Run("respeta el que envía el cliente", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/orders/x", nil)
		req.Header.Set(HeaderRequestID, "abc-123")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if got := rec.Header().Get(HeaderRequestID); got != "abc-123" {
			t.Errorf("X-Request-ID = %q", got)
		}
	})
}
