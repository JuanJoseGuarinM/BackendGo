package inventory

import (
	"errors"
	"testing"
)

func TestReserve(t *testing.T) {
	tests := []struct {
		name          string
		sku           string
		qty           int
		wantRemaining int
		wantErr       error
	}{
		{name: "reserva exitosa", sku: "SKU-1", qty: 3, wantRemaining: 7, wantErr: nil},
		{name: "reserva todo el stock", sku: "SKU-1", qty: 10, wantRemaining: 0, wantErr: nil},
		{name: "producto inexistente", sku: "SKU-9", qty: 1, wantRemaining: 0, wantErr: ErrProductNotFound},
		{name: "stock insuficiente", sku: "SKU-1", qty: 11, wantRemaining: 0, wantErr: ErrInsufficientStock},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stock := map[string]int{"SKU-1": 10}

			got, err := Reserve(stock, tt.sku, tt.qty)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, se esperaba %v", err, tt.wantErr)
			}
			if got != tt.wantRemaining {
				t.Errorf("restante = %d, se esperaba %d", got, tt.wantRemaining)
			}
		})
	}
}

func TestReserve_InvalidQty(t *testing.T) {
	stock := map[string]int{"SKU-1": 10}

	_, err := Reserve(stock, "SKU-1", 0)

	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("se esperaba *ValidationError, se obtuvo %v", err)
	}
	if valErr.Field != "qty" {
		t.Errorf("Field = %q, se esperaba %q", valErr.Field, "qty")
	}
	if stock["SKU-1"] != 10 {
		t.Errorf("el stock no debía cambiar, quedó en %d", stock["SKU-1"])
	}
}
