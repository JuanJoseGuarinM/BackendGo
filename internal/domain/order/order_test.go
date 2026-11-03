package order

import (
	"errors"
	"testing"
)

// Estas pruebas corren sin Gin y sin base de datos: solo reglas del negocio.

// mustItem es un helper de pruebas: crea un OrderItem o detiene la prueba.
func mustItem(t *testing.T, sku string, qty int, cents int64) OrderItem {
	t.Helper()
	it, err := NewOrderItem(sku, qty, cents)
	if err != nil {
		t.Fatalf("NewOrderItem: %v", err)
	}
	return it
}

func TestNewOrderItem(t *testing.T) {
	tests := []struct {
		name     string
		sku      string
		qty      int
		cents    int64
		wantErrs []error // todos deben encontrarse con errors.Is
	}{
		{name: "válido", sku: "A", qty: 1, cents: 100},
		{name: "sin SKU", sku: " ", qty: 1, cents: 100, wantErrs: []error{ErrInvalidOrder}},
		{name: "cantidad cero", sku: "A", qty: 0, cents: 100, wantErrs: []error{ErrInvalidOrder}},
		{name: "precio negativo", sku: "A", qty: 1, cents: -1, wantErrs: []error{ErrInvalidOrder, ErrNegativeMoney}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewOrderItem(tt.sku, tt.qty, tt.cents)

			if len(tt.wantErrs) == 0 && err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			for _, want := range tt.wantErrs {
				if !errors.Is(err, want) {
					t.Errorf("error = %v, debía contener %v", err, want)
				}
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		customerID string
		items      []OrderItem
		wantErr    error
	}{
		{name: "pedido válido", id: "o1", customerID: "c1", items: []OrderItem{mustItem(t, "A", 1, 100)}},
		{name: "pedido vacío se puede crear", id: "o1", customerID: "c1"},
		{name: "sin id", id: "", customerID: "c1", wantErr: ErrInvalidOrder},
		{name: "sin cliente", id: "o1", customerID: "", wantErr: ErrInvalidOrder},
		{name: "item sin constructor", id: "o1", customerID: "c1", items: []OrderItem{{}}, wantErr: ErrInvalidOrder},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, err := New(tt.id, tt.customerID, tt.items)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, se esperaba %v", err, tt.wantErr)
			}
			if err == nil && o.Status() != StatusPending {
				t.Errorf("estado = %s, se esperaba %s", o.Status(), StatusPending)
			}
		})
	}
}

func TestConfirm(t *testing.T) {
	item := mustItem(t, "A", 2, 100)

	tests := []struct {
		name       string
		order      *Order
		wantErr    error
		wantStatus Status
	}{
		{
			name:       "confirma pedido con productos",
			order:      Restore("o1", "c1", StatusPending, []OrderItem{item}),
			wantStatus: StatusConfirmed,
		},
		{
			name:       "no confirma pedido vacío y sigue PENDING",
			order:      Restore("o1", "c1", StatusPending, nil),
			wantErr:    ErrEmptyOrder,
			wantStatus: StatusPending,
		},
		{
			name:       "no confirma dos veces",
			order:      Restore("o1", "c1", StatusConfirmed, []OrderItem{item}),
			wantErr:    ErrAlreadyConfirmed,
			wantStatus: StatusConfirmed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.order.Confirm()

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, se esperaba %v", err, tt.wantErr)
			}
			if tt.order.Status() != tt.wantStatus {
				t.Errorf("estado = %s, se esperaba %s", tt.order.Status(), tt.wantStatus)
			}
		})
	}
}

func TestAddItem_ConfirmedOrderDoesNotChange(t *testing.T) {
	o := Restore("o1", "c1", StatusConfirmed, []OrderItem{mustItem(t, "A", 1, 100)})

	err := o.AddItem(mustItem(t, "B", 1, 100))

	if !errors.Is(err, ErrAlreadyConfirmed) {
		t.Fatalf("error = %v, se esperaba %v", err, ErrAlreadyConfirmed)
	}
	if len(o.Items()) != 1 {
		t.Errorf("el pedido confirmado cambió: %d productos", len(o.Items()))
	}
}

func TestItems_ReturnsCopy(t *testing.T) {
	o, _ := New("o1", "c1", []OrderItem{mustItem(t, "A", 1, 100)})

	items := o.Items()
	items[0] = mustItem(t, "HACK", 99, 0) // intento de modificar desde afuera

	if o.Items()[0].SKU() != "A" {
		t.Error("se pudo modificar el pedido sin pasar por sus métodos")
	}
}

func TestTotal(t *testing.T) {
	o, _ := New("o1", "c1", []OrderItem{
		mustItem(t, "A", 2, 1500),
		mustItem(t, "B", 1, 500),
	})

	if got := o.Total().Cents(); got != 3500 {
		t.Errorf("Total = %d, se esperaba 3500", got)
	}
}
