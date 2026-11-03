package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"orders-api/internal/adapters/classifier"
	"orders-api/internal/adapters/repository/memory"
	"orders-api/internal/application"
	"orders-api/internal/domain/order"
)

// newClassification guarda un pedido con qty unidades y arma el servicio con
// el "modelo" indicado y las reglas de respaldo.
func newClassification(t *testing.T, model application.OrderClassifier, qty int) *application.ClassificationService {
	t.Helper()
	repo := memory.NewOrderRepository()
	item, _ := order.NewOrderItem("SKU-1", qty, 1500)
	o, _ := order.New("o-1", "cliente-secreto", []order.OrderItem{item})
	_ = repo.Save(t.Context(), o)
	return application.NewClassificationService(repo, model, classifier.Rules{}, 50*time.Millisecond)
}

func TestClassify_ModelAnswer(t *testing.T) {
	c, err := newClassification(t, classifier.Simulated{}, 2).Classify(t.Context(), "o-1")

	if err != nil || c.Source != "model" || c.Label != application.LabelStandard {
		t.Errorf("Classify = %+v, %v", c, err)
	}
}

// La IA nunca rompe la API: en todos estos casos hay respuesta, de las reglas.
func TestClassify_FallsBackToRules(t *testing.T) {
	tests := []struct {
		name  string
		model application.OrderClassifier
		qty   int
	}{
		{name: "proveedor caído", model: classifier.Simulated{Mode: classifier.ModeFail}, qty: 2},
		{name: "alucina una etiqueta", model: classifier.Simulated{Mode: classifier.ModeInvalid}, qty: 2},
		{name: "tarda más que el timeout", model: classifier.Simulated{Mode: classifier.ModeSlow}, qty: 2},
		{name: "confianza baja", model: classifier.Simulated{}, qty: 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			c, err := newClassification(t, tt.model, tt.qty).Classify(t.Context(), "o-1")

			if err != nil {
				t.Fatalf("no debía fallar: %v", err)
			}
			if c.Source != "rules" {
				t.Errorf("source = %q, se esperaba rules", c.Source)
			}
			if time.Since(start) > time.Second {
				t.Errorf("tardó %v: el timeout no funcionó", time.Since(start))
			}
		})
	}
}

func TestClassify_OrderNotFound(t *testing.T) {
	_, err := newClassification(t, classifier.Simulated{}, 2).Classify(t.Context(), "nope")

	if !errors.Is(err, order.ErrNotFound) {
		t.Errorf("error = %v, se esperaba ErrNotFound", err)
	}
}

// spyClassifier guarda lo que recibió, para revisar qué datos se envían a la IA.
type spyClassifier struct {
	got *application.ClassificationInput
}

func (s spyClassifier) Classify(_ context.Context, in application.ClassificationInput) (application.Classification, error) {
	*s.got = in
	return application.Classification{Label: application.LabelStandard, Confidence: 0.9}, nil
}

func TestClassify_SendsOnlyNeededData(t *testing.T) {
	var got application.ClassificationInput
	_, _ = newClassification(t, spyClassifier{got: &got}, 3).Classify(t.Context(), "o-1")

	// Privacidad: el cliente ni siquiera existe en ClassificationInput, así que
	// "cliente-secreto" no puede llegar al modelo. Aquí verificamos lo que sí llega.
	if len(got.Items) != 1 || got.Items[0].Quantity != 3 || got.TotalCents != 4500 {
		t.Errorf("entrada al modelo = %+v", got)
	}
}
