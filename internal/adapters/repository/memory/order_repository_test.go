package memory

import (
	"context"
	"errors"
	"sync"
	"testing"

	"orders-api/internal/domain/order"
)

func newOrder(t *testing.T, id string) *order.Order {
	t.Helper()
	item, err := order.NewOrderItem("SKU-1", 2, 1500)
	if err != nil {
		t.Fatal(err)
	}
	o, err := order.New(id, "c1", []order.OrderItem{item})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestSaveAndFind_RoundTrip(t *testing.T) {
	repo := NewOrderRepository()
	o := newOrder(t, "o-1")

	if err := repo.Save(t.Context(), o); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.FindByID(t.Context(), "o-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}

	if got.ID() != "o-1" || got.CustomerID() != "c1" || got.Status() != order.StatusPending || got.Total().Cents() != 3000 {
		t.Errorf("pedido recuperado = id:%s cliente:%s estado:%s total:%d",
			got.ID(), got.CustomerID(), got.Status(), got.Total().Cents())
	}
}

func TestFindByID_NotFound(t *testing.T) {
	_, err := NewOrderRepository().FindByID(t.Context(), "nope")

	if !errors.Is(err, order.ErrNotFound) {
		t.Fatalf("error = %v, se esperaba ErrNotFound", err)
	}
}

// Cambiar el objeto recuperado NO cambia lo guardado hasta llamar Save,
// como pasaría con una base de datos real.
func TestFindByID_ReturnsIndependentCopy(t *testing.T) {
	repo := NewOrderRepository()
	_ = repo.Save(t.Context(), newOrder(t, "o-1"))

	got, _ := repo.FindByID(t.Context(), "o-1")
	_ = got.Confirm() // cambio en memoria, sin Save

	again, _ := repo.FindByID(t.Context(), "o-1")
	if again.Status() != order.StatusPending {
		t.Errorf("estado guardado = %s, no debía cambiar sin Save", again.Status())
	}
}

func TestRespectsCanceledContext(t *testing.T) {
	repo := NewOrderRepository()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if err := repo.Save(ctx, newOrder(t, "o-1")); !errors.Is(err, context.Canceled) {
		t.Errorf("Save con ctx cancelado = %v", err)
	}
	if _, err := repo.FindByID(ctx, "o-1"); !errors.Is(err, context.Canceled) {
		t.Errorf("FindByID con ctx cancelado = %v", err)
	}
}

// Muchas solicitudes a la vez (como en Gin). Con -race detecta si falta el mutex.
func TestConcurrentAccess(t *testing.T) {
	repo := NewOrderRepository()
	o := newOrder(t, "o-1")

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() { _ = repo.Save(t.Context(), o) })
		wg.Go(func() { _, _ = repo.FindByID(t.Context(), "o-1") })
	}
	wg.Wait()
}
