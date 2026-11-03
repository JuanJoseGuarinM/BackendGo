package main

import (
	"errors"
	"fmt"

	"orders-api/internal/practice/inventory"
)

func demoErrors() {
	stock := map[string]int{"SKU-1": 10}

	tryReserve(stock, "SKU-1", 3)  // éxito
	tryReserve(stock, "SKU-9", 1)  // producto no existe
	tryReserve(stock, "SKU-1", 50) // no alcanza el stock
	tryReserve(stock, "SKU-1", 0)  // cantidad inválida
}

func tryReserve(stock map[string]int, sku string, qty int) {
	fmt.Printf("\n> Reserve(%q, %d)\n", sku, qty)

	remaining, err := inventory.Reserve(stock, sku, qty)
	if err == nil {
		fmt.Printf("  OK, quedan %d\n", remaining)
		return
	}

	fmt.Printf("  error: %v\n", err)

	// Así decidirá la API (Jornada 2) qué código HTTP responder.
	var valErr *inventory.ValidationError
	switch {
	case errors.Is(err, inventory.ErrProductNotFound):
		fmt.Println("  errors.Is  -> es ErrProductNotFound   (en HTTP sería 404)")
	case errors.Is(err, inventory.ErrInsufficientStock):
		fmt.Println("  errors.Is  -> es ErrInsufficientStock (en HTTP sería 409)")
	case errors.As(err, &valErr):
		fmt.Printf("  errors.As  -> ValidationError, campo=%q (en HTTP sería 400)\n", valErr.Field)
	}
}
