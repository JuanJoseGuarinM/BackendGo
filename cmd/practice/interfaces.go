package main

import (
	"context"
	"fmt"
	"os"

	"orders-api/internal/practice/checkout"
	"orders-api/internal/practice/notify"
)

func demoInterfaces() {
	ctx := context.Background()

	// El MISMO servicio funciona con dos implementaciones distintas, porque
	// solo depende de la interfaz checkout.Notifier.
	fmt.Println("\n> Service con ConsoleNotifier")
	console := checkout.NewService(notify.ConsoleNotifier{Out: os.Stdout})
	_ = console.Complete(ctx, "ana")

	fmt.Println("\n> Service con PrefixNotifier (composición sobre MemoryNotifier)")
	prefixed := notify.PrefixNotifier{MemoryNotifier: &notify.MemoryNotifier{}, Prefix: "[URGENTE] "}
	_ = checkout.NewService(prefixed).Complete(ctx, "luis")
	fmt.Printf("  guardado en memoria: %+v\n", prefixed.Sent())

	fmt.Println("\n> Complete con cliente vacío")
	if err := console.Complete(ctx, ""); err != nil {
		fmt.Printf("  error: %v\n", err)
	}
}
