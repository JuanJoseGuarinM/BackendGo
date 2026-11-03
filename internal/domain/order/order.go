// Package order es el DOMINIO: las reglas del negocio sobre pedidos.
//
// Este paquete no importa Gin, ni HTTP, ni base de datos. Solo Go puro.
// Por eso sus reglas se pueden probar solas y no cambian si mañana cambiamos
// de framework web o de base de datos.
package order

import (
	"errors"
	"fmt"
	"slices"
)

// Status es el estado de un pedido.
type Status string

const (
	StatusPending   Status = "PENDING"
	StatusConfirmed Status = "CONFIRMED"
)

// Errores del dominio. La capa HTTP los traduce a códigos de estado,
// pero el dominio no sabe nada de HTTP.
var (
	ErrNotFound         = errors.New("pedido no encontrado")
	ErrInvalidOrder     = errors.New("pedido inválido")
	ErrEmptyOrder       = errors.New("el pedido no tiene productos")
	ErrAlreadyConfirmed = errors.New("el pedido ya está confirmado")
)

// Order es el AGREGADO (y su raíz, "aggregate root").
//
// Un agregado es un grupo de objetos que se tratan como UNA unidad:
// Order + sus OrderItem. Reglas:
//   - Desde afuera solo se habla con Order; nadie modifica un OrderItem
//     directamente.
//   - Order garantiza sus INVARIANTES (reglas que siempre deben cumplirse):
//     tiene cliente, sus productos son válidos, no se confirma vacío y no
//     cambia después de confirmado.
//   - Se guarda y se carga completo (un repositorio por agregado).
//
// Order es una ENTIDAD: tiene identidad (id). Dos pedidos con los mismos
// productos siguen siendo pedidos distintos si su id es distinto.
//
// Todos los campos son privados: la única forma de cambiar el estado es a
// través de sus métodos, que aplican las reglas.
type Order struct {
	id         string
	customerID string
	items      []OrderItem
	status     Status
}

// New crea un pedido válido en estado PENDING.
// Un pedido SÍ puede crearse vacío; lo que no puede es CONFIRMARSE vacío.
func New(id, customerID string, items []OrderItem) (*Order, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: el id es obligatorio", ErrInvalidOrder)
	}
	if customerID == "" {
		return nil, fmt.Errorf("%w: el cliente es obligatorio", ErrInvalidOrder)
	}
	for i, it := range items {
		if it.isZero() {
			return nil, fmt.Errorf("%w: el producto %d no fue creado con NewOrderItem", ErrInvalidOrder, i)
		}
	}

	return &Order{
		id:         id,
		customerID: customerID,
		items:      slices.Clone(items), // copia: nadie de afuera puede tocar nuestro slice
		status:     StatusPending,
	}, nil
}

// Restore reconstruye un pedido que YA existía (por ejemplo, leído de la base
// de datos). Solo deben usarlo los repositorios. No aplica las reglas de
// creación porque el pedido ya fue validado cuando se creó.
func Restore(id, customerID string, status Status, items []OrderItem) *Order {
	return &Order{
		id:         id,
		customerID: customerID,
		items:      slices.Clone(items),
		status:     status,
	}
}

// Getters: permiten LEER sin poder modificar.
func (o *Order) ID() string         { return o.id }
func (o *Order) CustomerID() string { return o.customerID }
func (o *Order) Status() Status     { return o.status }

// Items devuelve una COPIA: si alguien modifica el slice devuelto, el pedido
// no cambia.
func (o *Order) Items() []OrderItem { return slices.Clone(o.items) }

// AddItem agrega un producto. Invariante: un pedido confirmado ya no cambia.
func (o *Order) AddItem(item OrderItem) error {
	if o.status == StatusConfirmed {
		return ErrAlreadyConfirmed
	}
	if item.isZero() {
		return fmt.Errorf("%w: el producto no fue creado con NewOrderItem", ErrInvalidOrder)
	}
	o.items = append(o.items, item)
	return nil
}

// Confirm aplica la regla principal del negocio.
// Si alguna regla falla, el pedido NO cambia de estado.
func (o *Order) Confirm() error {
	if o.status == StatusConfirmed {
		return ErrAlreadyConfirmed
	}
	if len(o.items) == 0 {
		return ErrEmptyOrder
	}
	o.status = StatusConfirmed
	return nil
}

// Total suma los subtotales de todos los productos.
func (o *Order) Total() Money {
	var total Money
	for _, it := range o.items {
		total = total.Add(it.Subtotal())
	}
	return total
}
