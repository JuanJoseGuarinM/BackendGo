package order

import (
	"fmt"
	"strings"
)

// OrderItem es una línea del pedido: qué producto, cuántos y a qué precio.
//
// Solo existe DENTRO de un Order (el agregado): no tiene ID propio ni se
// guarda o consulta por separado. Se comporta como un value object.
//
// Los campos son privados: solo se crea con NewOrderItem, que valida.
type OrderItem struct {
	sku       string
	quantity  int
	unitPrice Money
}

// NewOrderItem crea una línea válida.
func NewOrderItem(sku string, quantity int, unitPriceCents int64) (OrderItem, error) {
	if strings.TrimSpace(sku) == "" {
		return OrderItem{}, fmt.Errorf("%w: el SKU es obligatorio", ErrInvalidOrder)
	}
	if quantity <= 0 {
		return OrderItem{}, fmt.Errorf("%w: la cantidad debe ser mayor que cero", ErrInvalidOrder)
	}
	price, err := NewMoney(unitPriceCents)
	if err != nil {
		// Doble %w: el error es a la vez ErrInvalidOrder y ErrNegativeMoney;
		// errors.Is encuentra los dos.
		return OrderItem{}, fmt.Errorf("%w: %w", ErrInvalidOrder, err)
	}
	return OrderItem{sku: sku, quantity: quantity, unitPrice: price}, nil
}

func (i OrderItem) SKU() string      { return i.sku }
func (i OrderItem) Quantity() int    { return i.quantity }
func (i OrderItem) UnitPrice() Money { return i.unitPrice }

// Subtotal = precio unitario × cantidad.
func (i OrderItem) Subtotal() Money {
	return i.unitPrice.Times(i.quantity)
}

// isZero detecta un OrderItem{} creado sin NewOrderItem. En Go cualquiera
// puede escribir OrderItem{} (el "valor cero"), así que el agregado lo revisa.
func (i OrderItem) isZero() bool {
	return i.sku == ""
}
