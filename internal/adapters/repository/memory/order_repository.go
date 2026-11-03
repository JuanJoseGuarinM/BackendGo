// Package memory guarda pedidos en un mapa en memoria.
// Es un ADAPTADOR: implementa application.OrderRepository sin mencionarlo
// (interfaces implícitas). Se pierde todo al reiniciar; sirve para aprender
// y para pruebas.
package memory

import (
	"context"
	"fmt"
	"sync"

	"orders-api/internal/domain/order"
)

// orderRecord es CÓMO se guarda un pedido: datos planos, como una fila de
// una base de datos. El repositorio traduce entre este formato y el objeto
// del dominio (que tiene campos privados y reglas).
//
// Así el dominio no sabe cómo se guarda, y el almacenamiento no sabe de reglas.
type orderRecord struct {
	ID         string
	CustomerID string
	Status     string
	Items      []itemRecord
}

type itemRecord struct {
	SKU            string
	Quantity       int
	UnitPriceCents int64
}

// OrderRepository es seguro para usar desde varias solicitudes HTTP a la vez
// (Gin atiende cada solicitud en su propia goroutine), por eso el mutex.
type OrderRepository struct {
	mu     sync.RWMutex // RWMutex: muchos pueden LEER a la vez; escribir es exclusivo
	orders map[string]orderRecord
}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{orders: make(map[string]orderRecord)}
}

func (r *OrderRepository) Save(ctx context.Context, o *order.Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	rec := toRecord(o)

	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[rec.ID] = rec
	return nil
}

func (r *OrderRepository) FindByID(ctx context.Context, id string) (*order.Order, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	rec, ok := r.orders[id]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("buscar pedido %q: %w", id, order.ErrNotFound)
	}
	return fromRecord(rec)
}

// toRecord: dominio → formato de almacenamiento.
func toRecord(o *order.Order) orderRecord {
	items := o.Items()
	rec := orderRecord{
		ID:         o.ID(),
		CustomerID: o.CustomerID(),
		Status:     string(o.Status()),
		Items:      make([]itemRecord, 0, len(items)),
	}
	for _, it := range items {
		rec.Items = append(rec.Items, itemRecord{
			SKU:            it.SKU(),
			Quantity:       it.Quantity(),
			UnitPriceCents: it.UnitPrice().Cents(),
		})
	}
	return rec
}

// fromRecord: formato de almacenamiento → dominio.
// Los items pasan otra vez por NewOrderItem: si hubiera un dato corrupto
// guardado, lo detectamos en vez de cargar un pedido inválido.
func fromRecord(rec orderRecord) (*order.Order, error) {
	items := make([]order.OrderItem, 0, len(rec.Items))
	for _, ir := range rec.Items {
		it, err := order.NewOrderItem(ir.SKU, ir.Quantity, ir.UnitPriceCents)
		if err != nil {
			// %v y NO %w, a propósito: es una falla INTERNA (dato corrupto), no
			// un error del usuario. Si envolviéramos ErrInvalidOrder, la API
			// respondería 422 culpando al cliente; así termina en 500.
			return nil, fmt.Errorf("pedido %q guardado con datos inválidos: %v", rec.ID, err)
		}
		items = append(items, it)
	}
	return order.Restore(rec.ID, rec.CustomerID, order.Status(rec.Status), items), nil
}
