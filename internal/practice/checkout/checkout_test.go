package checkout

import (
	"context"
	"errors"
	"testing"

	"orders-api/internal/practice/notify"
)

// Verificación en tiempo de compilación: si *notify.MemoryNotifier dejara de
// cumplir Notifier, el proyecto no compilaría. No cuesta nada en ejecución.
//
// Nota: notify.MemoryNotifier (sin *) NO cumple la interfaz, porque su método
// Notify tiene receptor por puntero.
var (
	_ Notifier = (*notify.MemoryNotifier)(nil)
	_ Notifier = notify.ConsoleNotifier{}
)

// failingNotifier es un "doble de prueba": simula un fallo del servicio de
// mensajes. Esto es posible gracias a que Service depende de una interfaz.
type failingNotifier struct{ err error }

func (f failingNotifier) Notify(context.Context, string, string) error { return f.err }

func TestComplete(t *testing.T) {
	errSMTP := errors.New("servidor de correo caído")

	tests := []struct {
		name     string
		customer string
		notifier Notifier
		wantErr  error
	}{
		{name: "compra exitosa", customer: "ana", notifier: &notify.MemoryNotifier{}, wantErr: nil},
		{name: "cliente vacío", customer: "", notifier: &notify.MemoryNotifier{}, wantErr: ErrEmptyCustomer},
		{name: "falla la notificación", customer: "ana", notifier: failingNotifier{err: errSMTP}, wantErr: errSMTP},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.notifier)

			err := svc.Complete(t.Context(), tt.customer)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, se esperaba %v", err, tt.wantErr)
			}
		})
	}
}

func TestComplete_SendsMessageToCustomer(t *testing.T) {
	mem := &notify.MemoryNotifier{}
	svc := NewService(mem)

	if err := svc.Complete(t.Context(), "ana"); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	sent := mem.Sent()
	if len(sent) != 1 || sent[0].To != "ana" {
		t.Errorf("enviados = %+v", sent)
	}
}
