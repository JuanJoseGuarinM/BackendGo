package concurrency

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunAll_AllTasksFinish(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	got, err := RunAll(ctx,
		SleepTask("catalogo", 20*time.Millisecond),
		SleepTask("precios", 10*time.Millisecond),
	)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	want := []string{"catalogo terminó", "precios terminó"}
	if !slices.Equal(got, want) {
		t.Errorf("resultados = %v, se esperaba %v", got, want)
	}
}

// El caso que pide el PDF: dos tareas concurrentes que se cancelan al vencer
// un context.WithTimeout.
func TestRunAll_TimeoutCancelsTasks(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := RunAll(ctx,
		SleepTask("catalogo", 2*time.Second),
		SleepTask("precios", 2*time.Second),
	)
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, se esperaba context.DeadlineExceeded", err)
	}
	// Si las tareas no respetaran el contexto, esto tardaría 2 segundos.
	if elapsed > 500*time.Millisecond {
		t.Errorf("tardó %v: las tareas no se cancelaron a tiempo", elapsed)
	}
}

func TestRunAll_FirstErrorCancelsTheOthers(t *testing.T) {
	errCatalog := errors.New("catálogo no disponible")
	var slowWasCanceled atomic.Bool

	failing := func(context.Context) (string, error) { return "", errCatalog }
	slow := func(ctx context.Context) (string, error) {
		select {
		case <-time.After(2 * time.Second):
			return "lenta terminó", nil
		case <-ctx.Done():
			slowWasCanceled.Store(true)
			return "", ctx.Err()
		}
	}

	_, err := RunAll(t.Context(), failing, slow)

	if !errors.Is(err, errCatalog) {
		t.Fatalf("error = %v, se esperaba %v", err, errCatalog)
	}
	// Como RunAll espera a todas las goroutines, al volver ya sabemos si la
	// tarea lenta vio la cancelación.
	if !slowWasCanceled.Load() {
		t.Error("la tarea lenta debió cancelarse cuando la otra falló")
	}
}

func TestCounter_ConcurrentInc(t *testing.T) {
	var c Counter
	const total = 1000

	var wg sync.WaitGroup
	for range total {
		wg.Go(c.Inc)
	}
	wg.Wait()

	if c.Value() != total {
		t.Errorf("Value = %d, se esperaba %d", c.Value(), total)
	}
}
