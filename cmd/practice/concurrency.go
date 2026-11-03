package main

import (
	"context"
	"fmt"
	"time"

	"orders-api/internal/practice/concurrency"
)

func demoConcurrency() {
	run("Timeout 1s, tareas de 300ms y 200ms (alcanzan)", time.Second,
		concurrency.SleepTask("catalogo", 300*time.Millisecond),
		concurrency.SleepTask("precios", 200*time.Millisecond),
	)

	run("Timeout 100ms, tareas de 2s (se cancelan)", 100*time.Millisecond,
		concurrency.SleepTask("catalogo", 2*time.Second),
		concurrency.SleepTask("precios", 2*time.Second),
	)
}

func run(title string, timeout time.Duration, tasks ...concurrency.Task) {
	fmt.Printf("\n> %s\n", title)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	results, err := concurrency.RunAll(ctx, tasks...)
	elapsed := time.Since(start).Round(10 * time.Millisecond)

	if err != nil {
		fmt.Printf("  error: %v (tardó %v)\n", err, elapsed)
		return
	}
	// Tardó ~300ms y no 500ms: las dos tareas corrieron EN PARALELO.
	fmt.Printf("  OK %v (tardó %v)\n", results, elapsed)
}
