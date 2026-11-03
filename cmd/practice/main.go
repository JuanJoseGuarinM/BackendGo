// Programa de demostración de la Jornada 1. Ejecuta: go run ./cmd/practice
//
// Cada archivo de esta carpeta muestra un tema:
//   - errors.go      → errores como valores, %w, errors.Is / errors.As
//   - interfaces.go  → interfaces, receptores, composición
//   - concurrency.go → goroutines, channels, context con timeout
package main

import "fmt"

func main() {
	section("1. Errores como valores")
	demoErrors()

	section("2. Interfaces y composición")
	demoInterfaces()

	section("3. Concurrencia con context")
	demoConcurrency()
}

func section(title string) {
	fmt.Printf("\n===== %s =====\n", title)
}
