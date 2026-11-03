package order

import (
	"errors"
	"testing"
)

func TestNewMoney(t *testing.T) {
	tests := []struct {
		name    string
		cents   int64
		wantErr error
	}{
		{name: "positivo", cents: 1500},
		{name: "cero es válido", cents: 0},
		{name: "negativo no existe", cents: -1, wantErr: ErrNegativeMoney},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMoney(tt.cents)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, se esperaba %v", err, tt.wantErr)
			}
			if err == nil && m.Cents() != tt.cents {
				t.Errorf("Cents = %d, se esperaba %d", m.Cents(), tt.cents)
			}
		})
	}
}

func TestMoney_IsImmutable(t *testing.T) {
	a, _ := NewMoney(1000)
	b, _ := NewMoney(500)

	sum := a.Add(b)

	if sum.Cents() != 1500 {
		t.Errorf("suma = %d", sum.Cents())
	}
	if a.Cents() != 1000 {
		t.Errorf("Add modificó el original: %d", a.Cents())
	}
}

func TestMoney_EqualByValue(t *testing.T) {
	a, _ := NewMoney(1500)
	b, _ := NewMoney(1500)

	// Value Object: se compara por valor, no por identidad.
	if a != b {
		t.Error("dos Money con el mismo valor deben ser iguales")
	}
}

func TestMoney_TimesNegativePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("se esperaba panic con n negativo")
		}
	}()
	m, _ := NewMoney(100)
	m.Times(-1)
}
