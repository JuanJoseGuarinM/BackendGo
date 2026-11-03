package notify

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestConsoleNotifier_WritesMessage(t *testing.T) {
	var buf bytes.Buffer // bytes.Buffer cumple io.Writer: no necesitamos la consola real
	n := ConsoleNotifier{Out: &buf}

	if err := n.Notify(t.Context(), "ana", "hola"); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !strings.Contains(buf.String(), "para ana: hola") {
		t.Errorf("salida = %q", buf.String())
	}
}

func TestMemoryNotifier_RespectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancelamos ANTES de llamar

	var n MemoryNotifier
	err := n.Notify(ctx, "ana", "hola")

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, se esperaba context.Canceled", err)
	}
	if len(n.Sent()) != 0 {
		t.Errorf("no debía guardar mensajes con el contexto cancelado")
	}
}

// Varias goroutines escriben a la vez. Sin el mutex, `go test -race` lo
// detectaría como condición de carrera y además se podrían perder mensajes.
func TestMemoryNotifier_ConcurrentNotify(t *testing.T) {
	var n MemoryNotifier
	const total = 100

	var wg sync.WaitGroup
	for range total {
		wg.Go(func() {
			_ = n.Notify(t.Context(), "ana", "hola")
		})
	}
	wg.Wait()

	if got := len(n.Sent()); got != total {
		t.Errorf("mensajes = %d, se esperaban %d", got, total)
	}
}

func TestPrefixNotifier_ComposesMemoryNotifier(t *testing.T) {
	p := PrefixNotifier{MemoryNotifier: &MemoryNotifier{}, Prefix: "[URGENTE] "}

	if err := p.Notify(t.Context(), "ana", "pedido listo"); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	// Sent() es un método de MemoryNotifier "promovido" por el embedding.
	sent := p.Sent()
	if len(sent) != 1 || sent[0].Body != "[URGENTE] pedido listo" {
		t.Errorf("enviados = %+v", sent)
	}
}
