// Package requestid guarda el identificador de correlación dentro del
// context.Context, para que viaje con la solicitud por todas las capas sin
// tener que pasarlo como parámetro en cada función.
//
// Es un paquete "de plataforma": no es dominio ni HTTP, lo puede usar
// cualquier capa (el middleware HTTP lo guarda, el cliente del catálogo y los
// logs lo leen).
package requestid

import "context"

// Header es el header HTTP donde viaja el ID entre servicios.
const Header = "X-Request-ID"

// ctxKey es un tipo privado: así ningún otro paquete puede pisar (por
// accidente) nuestro valor en el context usando la misma clave.
type ctxKey struct{}

// With devuelve un context hijo que lleva el ID.
func With(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// From lee el ID del context ("" si no hay).
func From(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}
