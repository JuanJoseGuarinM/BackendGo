// Package classifier tiene los ADAPTADORES de clasificación de pedidos:
//   - Rules:     reglas fijas (deterministas). Siempre responden lo mismo.
//   - Simulated: simula un modelo de IA (latencia, fallas, respuestas inválidas).
//
// Los dos cumplen application.OrderClassifier, así que se pueden intercambiar.
package classifier

import (
	"context"
	"fmt"

	"orders-api/internal/application"
)

// Umbrales del negocio. Al ser reglas fijas, se pueden explicar, probar y
// auditar exactamente. Esa es la ventaja de una regla sobre un modelo.
const (
	bulkUnits      = 20      // 20+ unidades en total → mayorista
	bulkTotalCents = 100_000 // $1.000,00 o más → mayorista
	reviewLineQty  = 50      // 50+ unidades de UN solo producto → inusual
)

// Rules clasifica con reglas deterministas. Es el respaldo cuando la IA falla.
type Rules struct{}

func (Rules) Classify(_ context.Context, in application.ClassificationInput) (application.Classification, error) {
	units, maxLine := 0, 0
	for _, it := range in.Items {
		units += it.Quantity
		maxLine = max(maxLine, it.Quantity)
	}

	switch {
	case maxLine >= reviewLineQty:
		return application.Classification{
			Label: application.LabelReview, Confidence: 1,
			Reason: fmt.Sprintf("una línea tiene %d unidades (umbral %d)", maxLine, reviewLineQty),
		}, nil
	case units >= bulkUnits || in.TotalCents >= bulkTotalCents:
		return application.Classification{
			Label: application.LabelBulk, Confidence: 1,
			Reason: fmt.Sprintf("%d unidades, total %d centavos", units, in.TotalCents),
		}, nil
	default:
		return application.Classification{
			Label: application.LabelStandard, Confidence: 1,
			Reason: "no supera ningún umbral",
		}, nil
	}
}
