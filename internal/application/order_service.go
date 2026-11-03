// Package application tiene los CASOS DE USO: "crear pedido", "consultar
// pedido", "confirmar pedido".
//
// Un caso de uso orquesta: busca, llama al dominio y guarda. NO tiene reglas
// de negocio propias (esas viven en domain) ni sabe de HTTP (eso es del
// adaptador).
package application

import (
	"context"
	"fmt"

	"orders-api/internal/domain/order"
)

// OrderRepository es lo que el caso de uso necesita para guardar pedidos.
// Se define AQUÍ (lado consumidor), igual que checkout.Notifier en la J1.
// Hoy la implementa un mapa en memoria; mañana podría ser PostgreSQL.
//
// Un repositorio por AGREGADO: guarda y carga el Order completo, con sus items.
type OrderRepository interface {
	Save(ctx context.Context, o *order.Order) error
	FindByID(ctx context.Context, id string) (*order.Order, error)
}

// CreateOrderInput son los datos para crear un pedido, en tipos simples.
// Es independiente del JSON que llegue por HTTP: el mismo caso de uso podría
// llamarse desde gRPC o desde una cola.
type CreateOrderInput struct {
	CustomerID string
	Items      []ItemInput
}

// ItemInput NO trae precio: el precio lo decide el catálogo, nunca el cliente.
type ItemInput struct {
	SKU      string
	Quantity int
}

// OrderService implementa los casos de uso de pedidos.
type OrderService struct {
	repo    OrderRepository
	catalog CatalogClient
	newID   func() string
}

// NewOrderService recibe sus dependencias (inyección). newID se inyecta para
// que las pruebas puedan usar IDs predecibles.
func NewOrderService(repo OrderRepository, catalog CatalogClient, newID func() string) *OrderService {
	return &OrderService{repo: repo, catalog: catalog, newID: newID}
}

func (s *OrderService) Create(ctx context.Context, in CreateOrderInput) (*order.Order, error) {
	items := make([]order.OrderItem, 0, len(in.Items))
	for i, it := range in.Items {
		// Llamada SÍNCRONA a otro servicio: si el catálogo falla, el pedido no
		// se crea. Le pasamos el ctx para que, si el cliente se va o vence el
		// tiempo, la llamada se cancele.
		p, err := s.catalog.GetProduct(ctx, it.SKU)
		if err != nil {
			return nil, fmt.Errorf("producto %d (%s): %w", i, it.SKU, err)
		}

		item, err := order.NewOrderItem(p.SKU, it.Quantity, p.PriceCents)
		if err != nil {
			return nil, fmt.Errorf("producto %d: %w", i, err)
		}
		items = append(items, item)
	}

	o, err := order.New(s.newID(), in.CustomerID, items)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, o); err != nil {
		return nil, fmt.Errorf("guardar pedido: %w", err)
	}
	return o, nil
}

func (s *OrderService) Get(ctx context.Context, id string) (*order.Order, error) {
	return s.repo.FindByID(ctx, id)
}

// Confirm: buscar → aplicar la regla del dominio → guardar.
// Si la regla falla, se devuelve el error y NO se guarda nada.
func (s *OrderService) Confirm(ctx context.Context, id string) (*order.Order, error) {
	o, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := o.Confirm(); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, o); err != nil {
		return nil, fmt.Errorf("guardar pedido %q: %w", id, err)
	}
	return o, nil
}
