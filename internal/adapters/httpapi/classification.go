package httpapi

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"orders-api/internal/application"
)

// Classifier es lo que el handler necesita (interfaz del lado consumidor).
type Classifier interface {
	Classify(ctx context.Context, orderID string) (application.Classification, error)
}

type classificationResponse struct {
	OrderID    string  `json:"order_id"`
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
	// Source dice si la respuesta vino del modelo o de las reglas de respaldo:
	// transparencia sobre decisiones tomadas por IA.
	Source string `json:"source"`
}

// GET /orders/:id/classification
//
// Endpoint SEPARADO, a propósito: crear o confirmar un pedido nunca espera
// a la IA. La clasificación se pide aparte y, si la IA falla, igual responde
// 200 con las reglas (nunca un 5xx por culpa del modelo).
func classify(svc Classifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		res, err := svc.Classify(c.Request.Context(), id)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, classificationResponse{
			OrderID:    id,
			Label:      res.Label,
			Confidence: res.Confidence,
			Reason:     res.Reason,
			Source:     res.Source,
		})
	}
}
