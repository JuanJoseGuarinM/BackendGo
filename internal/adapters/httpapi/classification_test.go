package httpapi

import (
	"net/http"
	"testing"

	"orders-api/internal/adapters/catalog"
	"orders-api/internal/adapters/classifier"
	"orders-api/internal/application"
)

func TestClassification_Endpoint(t *testing.T) {
	cat := catalog.NewStatic(catalog.SampleProducts()...)

	tests := []struct {
		name       string
		model      application.OrderClassifier
		wantSource string
	}{
		{name: "modelo responde", model: classifier.Simulated{}, wantSource: "model"},
		// Aunque la IA esté caída, la respuesta es 200: nunca un 5xx por el modelo.
		{name: "modelo caído → reglas", model: classifier.Simulated{Mode: classifier.ModeFail}, wantSource: "rules"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestRouterWith(cat, tt.model)
			do(r, http.MethodPost, "/orders", validOrder) // crea o-1

			rec := do(r, http.MethodGet, "/orders/o-1/classification", "")

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d\n%s", rec.Code, rec.Body.String())
			}
			got := decode[classificationResponse](t, rec)
			if got.Source != tt.wantSource || got.Label != application.LabelStandard || got.OrderID != "o-1" {
				t.Errorf("respuesta = %+v", got)
			}
		})
	}
}

func TestClassification_OrderNotFound(t *testing.T) {
	rec := do(newTestRouter(), http.MethodGet, "/orders/nope/classification", "")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}
