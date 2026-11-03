package inventory

import (
	"errors"
	"fmt"
)

// Errores centinela: fallos conocidos que se pueden reconocer con errors.Is.
var (
	ErrProductNotFound   = errors.New("producto no encontrado")
	ErrInsufficientStock = errors.New("stock insuficiente")
)

// ValidationError es un error con datos: dice qué campo falló y por qué.
// Se reconoce con errors.As.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("campo %q inválido: %s", e.Field, e.Reason)
}

// Reserve descuenta qty unidades del producto sku y devuelve el stock restante.
func Reserve(stock map[string]int, sku string, qty int) (int, error) {
	if qty <= 0 {
		return 0, &ValidationError{Field: "qty", Reason: "debe ser mayor que cero"}
	}

	available, ok := stock[sku]
	if !ok {
		return 0, fmt.Errorf("reservar %q: %w", sku, ErrProductNotFound)
	}

	if available < qty {
		return 0, fmt.Errorf("reservar %q (disponible %d, pedido %d): %w", sku, available, qty, ErrInsufficientStock)
	}

	stock[sku] = available - qty
	return stock[sku], nil
}
