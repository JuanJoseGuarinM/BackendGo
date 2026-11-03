package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"orders-api/internal/application"
	"orders-api/internal/domain/order"
)

// writeError traduce un error del dominio/aplicación a una respuesta HTTP.
//
// Es el ÚNICO lugar donde se decide qué código HTTP corresponde a cada error.
// El dominio no sabe de HTTP; aquí, en el borde, hacemos la traducción.
func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, order.ErrNotFound):
		respondError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "El pedido no existe")
	case errors.Is(err, order.ErrAlreadyConfirmed):
		respondError(c, http.StatusConflict, "ORDER_ALREADY_CONFIRMED", "El pedido ya fue confirmado")
	case errors.Is(err, order.ErrEmptyOrder):
		respondError(c, http.StatusUnprocessableEntity, "ORDER_EMPTY", "El pedido debe incluir productos")
	case errors.Is(err, order.ErrInvalidOrder):
		respondError(c, http.StatusUnprocessableEntity, "INVALID_ORDER", err.Error())
	case errors.Is(err, application.ErrProductNotFound):
		respondError(c, http.StatusUnprocessableEntity, "PRODUCT_NOT_FOUND", "Uno de los productos no existe en el catálogo")
	case errors.Is(err, application.ErrCatalogUnavailable):
		// 503: no es culpa del cliente ni un bug nuestro; una dependencia está
		// caída. El cliente puede reintentar más tarde.
		// *Context: el correlation_id lo agrega el logger desde el ctx (J6).
		slog.WarnContext(c.Request.Context(), "catálogo no disponible", "error", err)
		respondError(c, http.StatusServiceUnavailable, "CATALOG_UNAVAILABLE", "El catálogo no está disponible, intenta más tarde")
	default:
		// Error inesperado: lo registramos completo en el log (para nosotros),
		// pero al cliente solo le damos un mensaje genérico. Nunca exponer
		// detalles internos (rutas, SQL, stack traces...).
		slog.ErrorContext(c.Request.Context(), "error inesperado",
			"error", err,
			"route", c.FullPath(),
		)
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Ocurrió un error inesperado")
	}
}

// writeBadRequest responde 400 cuando el JSON está mal formado o no cumple
// la forma esperada. Es un error de TRANSPORTE, no del negocio.
func writeBadRequest(c *gin.Context) {
	respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "El cuerpo de la solicitud no es válido")
}

func respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, errorResponse{
		Code:          code,
		Message:       message,
		CorrelationID: correlationID(c),
	})
}
