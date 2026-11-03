// Package httpapi es el ADAPTADOR HTTP: traduce HTTP ↔ casos de uso.
//
// Responsabilidades de un handler (y nada más):
//  1. Leer y validar la FORMA del request (JSON, parámetros).
//  2. Llamar al caso de uso.
//  3. Traducir el resultado o el error a una respuesta HTTP.
//
// Las reglas del negocio NO van aquí: Gin es solo la "puerta de entrada".
package httpapi

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"orders-api/internal/application"
	"orders-api/internal/domain/order"
)

// OrderService es lo que los handlers necesitan (interfaz del lado consumidor).
// *application.OrderService la cumple; en pruebas podríamos pasar un falso.
type OrderService interface {
	Create(ctx context.Context, in application.CreateOrderInput) (*order.Order, error)
	Get(ctx context.Context, id string) (*order.Order, error)
	Confirm(ctx context.Context, id string) (*order.Order, error)
}

type orderHandler struct {
	svc OrderService
}

// POST /orders
func (h *orderHandler) create(c *gin.Context) {
	var req createOrderRequest
	// ShouldBindJSON lee el JSON y aplica las etiquetas `binding`.
	// (BindJSON respondería 400 por su cuenta; Should* nos deja controlar
	// el formato de la respuesta).
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBadRequest(c)
		return
	}

	// c.Request.Context() es el context.Context de la solicitud: si el
	// cliente se desconecta, se cancela. Ese es el que se pasa hacia adentro.
	// *gin.Context NO se pasa a las capas internas: es cosa de Gin.
	o, err := h.svc.Create(c.Request.Context(), req.toInput())
	if err != nil {
		writeError(c, err)
		return
	}

	c.Header("Location", "/orders/"+o.ID()) // buena práctica REST al crear
	c.JSON(http.StatusCreated, toOrderResponse(o))
}

// GET /orders/:id
func (h *orderHandler) get(c *gin.Context) {
	o, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toOrderResponse(o))
}

// POST /orders/:id/confirm
// Es POST y no PUT porque es una ACCIÓN que cambia el estado, no el reemplazo
// del recurso completo.
func (h *orderHandler) confirm(c *gin.Context) {
	o, err := h.svc.Confirm(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toOrderResponse(o))
}
