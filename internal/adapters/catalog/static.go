package catalog

import (
	"context"
	"fmt"

	"orders-api/internal/application"
)

// Static es un catálogo EN MEMORIA. Cumple el mismo puerto que HTTPClient,
// así que la aplicación no nota la diferencia. Se usa en pruebas y para
// correr la API sola, sin levantar el catálogo.
type Static struct {
	products map[string]application.Product
}

func NewStatic(products ...application.Product) *Static {
	m := make(map[string]application.Product, len(products))
	for _, p := range products {
		m[p.SKU] = p
	}
	return &Static{products: m}
}

func (s *Static) GetProduct(ctx context.Context, sku string) (application.Product, error) {
	if err := ctx.Err(); err != nil {
		return application.Product{}, err
	}
	p, ok := s.products[sku]
	if !ok {
		return application.Product{}, fmt.Errorf("sku %q: %w", sku, application.ErrProductNotFound)
	}
	return p, nil
}

// SampleProducts son productos ficticios de ejemplo (los usa también cmd/catalog).
func SampleProducts() []application.Product {
	return []application.Product{
		{SKU: "SKU-1", Name: "Café 500 g", PriceCents: 1500},
		{SKU: "SKU-2", Name: "Arepas x10", PriceCents: 800},
		{SKU: "SKU-3", Name: "Chocolatina", PriceCents: 500},
	}
}
