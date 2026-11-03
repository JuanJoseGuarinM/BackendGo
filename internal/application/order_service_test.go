package application_test

// Paquete application_test (con _test): las pruebas usan la aplicación DESDE
// AFUERA, como lo haría un adaptador. Además así podemos importar
// adapters/catalog, que a su vez importa application, sin un ciclo.

import (
	"context"
	"errors"
	"testing"

	"orders-api/internal/adapters/catalog"
	"orders-api/internal/adapters/repository/memory"
	"orders-api/internal/application"
	"orders-api/internal/domain/order"
)

func newService(cat application.CatalogClient) *application.OrderService {
	if cat == nil {
		cat = catalog.NewStatic(catalog.SampleProducts()...)
	}
	return application.NewOrderService(memory.NewOrderRepository(), cat, func() string { return "o-1" })
}

func TestCreate_UsesCatalogPrice(t *testing.T) {
	svc := newService(nil)

	o, err := svc.Create(t.Context(), application.CreateOrderInput{
		CustomerID: "c1",
		Items:      []application.ItemInput{{SKU: "SKU-1", Quantity: 2}},
	})

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if o.Total().Cents() != 3000 { // 2 × 1500 del catálogo
		t.Errorf("total = %d, se esperaba 3000", o.Total().Cents())
	}
}

func TestCreate_UnknownProductIsNotSaved(t *testing.T) {
	svc := newService(nil)

	_, err := svc.Create(t.Context(), application.CreateOrderInput{
		CustomerID: "c1",
		Items:      []application.ItemInput{{SKU: "NOPE", Quantity: 1}},
	})

	if !errors.Is(err, application.ErrProductNotFound) {
		t.Fatalf("error = %v, se esperaba %v", err, application.ErrProductNotFound)
	}
	if _, err := svc.Get(t.Context(), "o-1"); !errors.Is(err, order.ErrNotFound) {
		t.Errorf("no debía guardarse el pedido")
	}
}

type downCatalog struct{}

func (downCatalog) GetProduct(context.Context, string) (application.Product, error) {
	return application.Product{}, application.ErrCatalogUnavailable
}

func TestCreate_CatalogDownIsNotSaved(t *testing.T) {
	svc := newService(downCatalog{})

	_, err := svc.Create(t.Context(), application.CreateOrderInput{
		CustomerID: "c1",
		Items:      []application.ItemInput{{SKU: "SKU-1", Quantity: 1}},
	})

	if !errors.Is(err, application.ErrCatalogUnavailable) {
		t.Fatalf("error = %v, se esperaba %v", err, application.ErrCatalogUnavailable)
	}
	if _, err := svc.Get(t.Context(), "o-1"); !errors.Is(err, order.ErrNotFound) {
		t.Errorf("con el catálogo caído no debía guardarse nada")
	}
}

// Evidencia del PDF (J3): el pedido no pasa a confirmado si incumple la regla,
// y el error se propaga hasta quien llama.
func TestConfirm_EmptyOrderStaysPending(t *testing.T) {
	svc := newService(nil)
	if _, err := svc.Create(t.Context(), application.CreateOrderInput{CustomerID: "c1"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err := svc.Confirm(t.Context(), "o-1")

	if !errors.Is(err, order.ErrEmptyOrder) {
		t.Fatalf("error = %v, se esperaba %v", err, order.ErrEmptyOrder)
	}
	saved, _ := svc.Get(t.Context(), "o-1")
	if saved.Status() != order.StatusPending {
		t.Errorf("estado guardado = %s, se esperaba PENDING", saved.Status())
	}
}

// failingSaveRepo simula una base de datos que falla al guardar.
type failingSaveRepo struct {
	*memory.OrderRepository // composición: hereda FindByID, redefine Save
	err                     error
}

func (r failingSaveRepo) Save(context.Context, *order.Order) error { return r.err }

func TestConfirm_RepositoryErrorIsWrapped(t *testing.T) {
	mem := memory.NewOrderRepository()
	item, _ := order.NewOrderItem("A", 1, 100)
	o, _ := order.New("o-1", "c1", []order.OrderItem{item})
	_ = mem.Save(t.Context(), o)

	errDB := errors.New("base de datos caída")
	svc := application.NewOrderService(failingSaveRepo{OrderRepository: mem, err: errDB}, nil, nil)

	_, err := svc.Confirm(t.Context(), "o-1")

	if !errors.Is(err, errDB) {
		t.Fatalf("error = %v, debía envolver %v", err, errDB)
	}
}
