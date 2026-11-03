// Package notify tiene implementaciones CONCRETAS para enviar mensajes.
//
// Fíjate que este paquete NO declara ninguna interfaz Notifier. En Go las
// interfaces se cumplen de forma implícita: si un tipo tiene el método
// Notify(ctx, to, body) error, ya sirve para cualquier interfaz que pida eso
// (por ejemplo checkout.Notifier), sin escribir "implements".
package notify

import (
	"context"
	"fmt"
	"io"
	"slices"
	"sync"
)

// Message es un mensaje enviado.
type Message struct {
	To   string
	Body string
}

// ConsoleNotifier escribe los mensajes en un io.Writer (por ejemplo os.Stdout).
//
// Receptor POR VALOR (c ConsoleNotifier): el método no modifica el struct y el
// struct es pequeño, así que copiarlo no cuesta nada.
type ConsoleNotifier struct {
	Out io.Writer
}

func (c ConsoleNotifier) Notify(ctx context.Context, to, body string) error {
	// Buena práctica: si quien llama ya canceló, no hacemos el trabajo.
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := fmt.Fprintf(c.Out, "  [consola] para %s: %s\n", to, body)
	return err
}

// MemoryNotifier guarda los mensajes en memoria. Útil en pruebas.
//
// Receptor POR PUNTERO (m *MemoryNotifier) por dos razones:
//  1. Notify MODIFICA el struct (agrega a sent). Con receptor por valor
//     modificaría una copia y el cambio se perdería.
//  2. Contiene un sync.Mutex, y un mutex NUNCA debe copiarse.
//
// Consecuencia: el que cumple la interfaz es *MemoryNotifier (el puntero),
// no MemoryNotifier (el valor).
type MemoryNotifier struct {
	mu   sync.Mutex // protege sent: varias goroutines pueden llamar Notify a la vez
	sent []Message
}

func (m *MemoryNotifier) Notify(ctx context.Context, to, body string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock() // defer: se libera el lock al salir, pase lo que pase
	m.sent = append(m.sent, Message{To: to, Body: body})
	return nil
}

// Sent devuelve una COPIA de los mensajes enviados, para que quien llama no
// pueda modificar el slice interno sin pasar por el mutex.
func (m *MemoryNotifier) Sent() []Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.sent)
}

// PrefixNotifier muestra COMPOSICIÓN con embedding (Go no tiene herencia).
//
// Al incrustar *MemoryNotifier (campo sin nombre), sus métodos se "promueven":
// se puede llamar p.Sent() directamente. PrefixNotifier redefine Notify para
// agregar un prefijo y luego delega en el MemoryNotifier incrustado.
type PrefixNotifier struct {
	*MemoryNotifier
	Prefix string
}

func (p PrefixNotifier) Notify(ctx context.Context, to, body string) error {
	return p.MemoryNotifier.Notify(ctx, to, p.Prefix+body)
}
