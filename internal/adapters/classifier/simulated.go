package classifier

import (
	"context"
	"errors"
	"fmt"
	"time"

	"orders-api/internal/application"
)

// Modos del modelo simulado, para practicar los riesgos reales de la IA.
const (
	ModeOK      = ""        // responde bien
	ModeFail    = "fail"    // el proveedor está caído
	ModeInvalid = "invalid" // "alucina": inventa una etiqueta que no existe
	ModeSlow    = "slow"    // tarda más que el timeout
)

// Simulated imita un modelo de IA SIN contratar un proveedor (lo pide el PDF).
//
// Un modelo real se llamaría por HTTP a un proveedor aprobado, con una API key
// guardada en un gestor de secretos. Este adaptador simula su comportamiento:
// tarda, da una CONFIANZA (no una certeza), explica en lenguaje natural y a
// veces falla o responde mal.
type Simulated struct {
	Mode    string
	Latency time.Duration // latencia normal de la "inferencia"
}

var errModelUnavailable = errors.New("modelo simulado: proveedor no disponible")

func (s Simulated) Classify(ctx context.Context, in application.ClassificationInput) (application.Classification, error) {
	latency := s.Latency
	if s.Mode == ModeSlow {
		latency = 10 * time.Second
	}
	if err := wait(ctx, latency); err != nil {
		return application.Classification{}, err // timeout: el servicio dejó de esperar
	}

	switch s.Mode {
	case ModeFail:
		return application.Classification{}, errModelUnavailable
	case ModeInvalid:
		return application.Classification{Label: "URGENTE_VIP", Confidence: 0.99, Reason: "suena importante"}, nil
	}

	// "Inferencia" simulada: a diferencia de Rules, el modelo no da certeza
	// sino una confianza que baja cuando el caso es ambiguo.
	units := 0
	for _, it := range in.Items {
		units += it.Quantity
	}
	switch {
	case units >= 30:
		return application.Classification{
			Label: application.LabelBulk, Confidence: 0.93,
			Reason: fmt.Sprintf("Por el volumen (%d unidades) parece una compra mayorista.", units),
		}, nil
	case units >= 15:
		// Zona gris: el modelo "duda". Con confianza < 0.6 el servicio usará las reglas.
		return application.Classification{
			Label: application.LabelBulk, Confidence: 0.55,
			Reason: fmt.Sprintf("Con %d unidades podría ser mayorista, pero no es claro.", units),
		}, nil
	default:
		return application.Classification{
			Label: application.LabelStandard, Confidence: 0.91,
			Reason: fmt.Sprintf("Pedido de %d unidades, similar a una compra habitual.", units),
		}, nil
	}
}

func wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
