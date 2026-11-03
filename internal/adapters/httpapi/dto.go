package httpapi

import (
	"orders-api/internal/application"
	"orders-api/internal/domain/order"
)

// DTO (Data Transfer Object): la forma del JSON que entra y sale por HTTP.
//
// Son tipos SEPARADOS del dominio a propósito:
//   - El contrato público de la API no cambia si cambiamos el dominio.
//   - Las etiquetas `json` y `binding` son cosas de transporte; no deben
//     ensuciar el dominio.

// createOrderRequest es el cuerpo de POST /orders.
// Las etiquetas `binding` validan la FORMA del request (Gin usa la librería
// validator). Las reglas del NEGOCIO las valida el dominio.
type createOrderRequest struct {
	CustomerID string        `json:"customer_id" binding:"required"`
	Items      []itemRequest `json:"items" binding:"dive"` // dive: valida cada item
}

// itemRequest NO tiene precio (J4): el precio lo trae el servidor desde el
// catálogo. Si el cliente pudiera enviarlo, podría comprar a $0.
type itemRequest struct {
	SKU      string `json:"sku" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,gt=0"`
}

type orderResponse struct {
	ID         string         `json:"id"`
	CustomerID string         `json:"customer_id"`
	Status     string         `json:"status"`
	Items      []itemResponse `json:"items"`
	Total      int64          `json:"total"`
}

type itemResponse struct {
	SKU       string `json:"sku"`
	Quantity  int    `json:"quantity"`
	UnitPrice int64  `json:"unit_price"`
}

// errorResponse es el formato ÚNICO de error de toda la API.
type errorResponse struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

// toInput: DTO de HTTP → entrada del caso de uso.
func (r createOrderRequest) toInput() application.CreateOrderInput {
	items := make([]application.ItemInput, 0, len(r.Items))
	for _, it := range r.Items {
		items = append(items, application.ItemInput{SKU: it.SKU, Quantity: it.Quantity})
	}
	return application.CreateOrderInput{CustomerID: r.CustomerID, Items: items}
}

// toOrderResponse: objeto del dominio → DTO de salida. Usa los getters
// porque los campos del dominio son privados.
func toOrderResponse(o *order.Order) orderResponse {
	domainItems := o.Items()
	items := make([]itemResponse, 0, len(domainItems)) // make(..., 0): JSON muestra [] y no null
	for _, it := range domainItems {
		items = append(items, itemResponse{SKU: it.SKU(), Quantity: it.Quantity(), UnitPrice: it.UnitPrice().Cents()})
	}
	return orderResponse{
		ID:         o.ID(),
		CustomerID: o.CustomerID(),
		Status:     string(o.Status()),
		Items:      items,
		Total:      o.Total().Cents(),
	}
}
