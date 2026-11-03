// Package concurrency muestra goroutines, channels, context y mutex.
//
// Dos formas de coordinar goroutines en Go:
//   - Comunicarse por CHANNELS (RunAll): cada goroutine envía su resultado por
//     un canal y una sola goroutine los recibe. Nadie comparte variables.
//   - Compartir MEMORIA protegida con un MUTEX (Counter): varias goroutines
//     tocan la misma variable, pero de a una a la vez.
//
// Lema de Go: "No te comuniques compartiendo memoria; comparte memoria
// comunicándote". Se prefieren los channels para pasar datos entre tareas, y
// el mutex para proteger un estado simple (un contador, un mapa, un caché).
package concurrency

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Task es una tarea que respeta la cancelación del contexto.
type Task func(ctx context.Context) (string, error)

// RunAll ejecuta todas las tareas EN PARALELO y devuelve sus resultados en el
// mismo orden en que se pasaron las tareas (no en el orden en que terminaron).
//
// Garantías (lo que se sustenta en la entrevista):
//   - Si una tarea falla, se cancela el contexto y las demás se detienen.
//   - Si el ctx de quien llama vence (timeout), todas se detienen.
//   - RunAll NO retorna hasta que todas las goroutines terminaron: no deja
//     goroutines "huérfanas" (goroutine leak).
func RunAll(ctx context.Context, tasks ...Task) ([]string, error) {
	// Contexto hijo: cancelarlo avisa a todas las tareas. El defer asegura que
	// se liberen sus recursos siempre.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		index int
		value string
		err   error
	}

	// Canal con buffer del tamaño de las tareas: cada goroutine puede enviar su
	// resultado sin quedarse bloqueada esperando a que alguien lo lea.
	results := make(chan result, len(tasks))

	var wg sync.WaitGroup
	for i, task := range tasks {
		wg.Go(func() { // wg.Go lanza la goroutine y la registra en el WaitGroup
			value, err := task(ctx)
			results <- result{index: i, value: value, err: err}
		})
	}

	values := make([]string, len(tasks))
	var firstErr error

	// Recibimos exactamente un resultado por tarea. Aunque haya error, seguimos
	// recibiendo para que todas terminen.
	for range len(tasks) {
		r := <-results
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
				cancel() // primera falla: avisamos a las demás que paren
			}
			continue
		}
		values[r.index] = r.value
	}
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	return values, nil
}

// SleepTask crea una tarea que "trabaja" durante d, pero se detiene apenas el
// contexto se cancela. Simula, por ejemplo, una llamada a otro servicio.
func SleepTask(name string, d time.Duration) Task {
	return func(ctx context.Context) (string, error) {
		timer := time.NewTimer(d)
		defer timer.Stop()

		// select espera lo que pase PRIMERO: que termine el trabajo o que
		// cancelen el contexto.
		select {
		case <-timer.C:
			return name + " terminó", nil
		case <-ctx.Done():
			return "", fmt.Errorf("%s cancelada: %w", name, ctx.Err())
		}
	}
}

// Counter es un contador seguro para usar desde muchas goroutines.
//
// Sin el mutex, "c.n++" desde varias goroutines es una CONDICIÓN DE CARRERA:
// n++ es leer, sumar y escribir; dos goroutines pueden leer el mismo valor y
// se pierde un incremento. `go test -race` detecta este tipo de error.
type Counter struct {
	mu sync.Mutex
	n  int
}

func (c *Counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
}

func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}
