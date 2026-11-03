package application

import (
	"context"
	"errors"
)

// Errores del PUERTO del catálogo. Los define la aplicación (quien usa el
// catálogo); el adaptador traduce sus fallas HTTP a estos errores. Así la
// aplicación no sabe si el catálogo es HTTP, gRPC o un mapa en memoria.
var (
	ErrProductNotFound    = errors.New("el producto no existe en el catálogo")
	ErrCatalogUnavailable = errors.New("el catálogo no está disponible")
)

// Product es lo que el catálogo nos dice de un producto.
//
// Es un tipo NUESTRO, no el JSON del catálogo: el catálogo es otro "bounded
// context" (J3) y no queremos que su modelo se meta en el nuestro. El
// adaptador traduce.
type Product struct {
	SKU        string
	Name       string
	PriceCents int64
}

// CatalogClient es el puerto hacia el catálogo externo (lado consumidor).
type CatalogClient interface {
	GetProduct(ctx context.Context, sku string) (Product, error)
}
