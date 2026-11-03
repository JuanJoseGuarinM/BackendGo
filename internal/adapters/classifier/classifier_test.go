package classifier

import (
	"context"
	"errors"
	"testing"
	"time"

	"orders-api/internal/application"
)

var (
	_ application.OrderClassifier = Rules{}
	_ application.OrderClassifier = Simulated{}
)

func input(qtys ...int) application.ClassificationInput {
	in := application.ClassificationInput{}
	for _, q := range qtys {
		in.Items = append(in.Items, application.ClassificationItem{SKU: "SKU-1", Quantity: q, UnitPriceCents: 100})
		in.TotalCents += int64(q) * 100
	}
	return in
}

func TestRules(t *testing.T) {
	tests := []struct {
		name string
		in   application.ClassificationInput
		want string
	}{
		{name: "pedido chico", in: input(2, 3), want: application.LabelStandard},
		{name: "20 unidades en total", in: input(10, 10), want: application.LabelBulk},
		{name: "total alto", in: application.ClassificationInput{Items: []application.ClassificationItem{{Quantity: 1}}, TotalCents: 100_000}, want: application.LabelBulk},
		{name: "una línea enorme", in: input(60), want: application.LabelReview},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := Rules{}.Classify(t.Context(), tt.in)

			if err != nil || c.Label != tt.want || c.Confidence != 1 {
				t.Errorf("Classify = %+v, %v; se esperaba %s con confianza 1", c, err, tt.want)
			}
		})
	}
}

func TestSimulated_Modes(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		c, err := Simulated{}.Classify(t.Context(), input(2))
		if err != nil || c.Label != application.LabelStandard || c.Confidence < 0.6 {
			t.Errorf("Classify = %+v, %v", c, err)
		}
	})

	t.Run("caso ambiguo: confianza baja", func(t *testing.T) {
		c, _ := Simulated{}.Classify(t.Context(), input(16))
		if c.Confidence >= 0.6 {
			t.Errorf("confianza = %v, se esperaba < 0.6", c.Confidence)
		}
	})

	t.Run("fail", func(t *testing.T) {
		_, err := Simulated{Mode: ModeFail}.Classify(t.Context(), input(2))
		if !errors.Is(err, errModelUnavailable) {
			t.Errorf("error = %v", err)
		}
	})

	t.Run("invalid: alucina una etiqueta", func(t *testing.T) {
		c, _ := Simulated{Mode: ModeInvalid}.Classify(t.Context(), input(2))
		if c.Label == application.LabelStandard || c.Label == application.LabelBulk || c.Label == application.LabelReview {
			t.Errorf("se esperaba una etiqueta inválida, llegó %q", c.Label)
		}
	})

	t.Run("slow: respeta el timeout del context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, err := Simulated{Mode: ModeSlow}.Classify(ctx, input(2))

		if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > time.Second {
			t.Errorf("error = %v tras %v", err, time.Since(start))
		}
	})
}
