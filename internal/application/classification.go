package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"orders-api/internal/domain/order"
)

// Clasificación de pedidos con IA (J7).
//
// Ideas clave de diseño:
//  1. La IA está AISLADA detrás de una interfaz (OrderClassifier), igual que
//     el catálogo. El dominio no sabe que existe; cambiar de proveedor de IA
//     es cambiar un adaptador.
//  2. La IA es un COMPLEMENTO, no una regla del negocio: confirmar un pedido
//     nunca depende de ella. Si falla, la API sigue funcionando.
//  3. Nunca se confía ciegamente en la respuesta: se VALIDA (etiqueta
//     permitida, confianza en rango) y, si no sirve, se usan reglas fijas.

// Etiquetas permitidas. Un modelo puede "alucinar" una etiqueta que no existe;
// por eso la lista la define el sistema, no el modelo.
const (
	LabelStandard = "STANDARD" // pedido normal
	LabelBulk     = "BULK"     // pedido grande / mayorista
	LabelReview   = "REVIEW"   // inusual: conviene que una persona lo revise
)

var allowedLabels = map[string]bool{LabelStandard: true, LabelBulk: true, LabelReview: true}

// ClassificationInput es lo que se le envía al clasificador.
//
// MINIMIZACIÓN DE DATOS (privacidad): NO incluye el cliente. El modelo solo
// recibe lo necesario para clasificar. Al no existir el campo, es imposible
// enviarlo por error a un proveedor externo.
type ClassificationInput struct {
	Items      []ClassificationItem
	TotalCents int64
}

type ClassificationItem struct {
	SKU            string
	Quantity       int
	UnitPriceCents int64
}

// Classification es el resultado.
type Classification struct {
	Label      string
	Confidence float64 // 0 a 1: qué tan seguro está
	Reason     string  // explicación legible (útil para auditar decisiones)
	Source     string  // "model" o "rules": de dónde salió la respuesta
}

// OrderClassifier es el PUERTO hacia la IA (lado consumidor).
type OrderClassifier interface {
	Classify(ctx context.Context, in ClassificationInput) (Classification, error)
}

var errInvalidClassification = errors.New("respuesta del clasificador inválida")

// ClassificationService es el caso de uso "clasificar un pedido".
type ClassificationService struct {
	repo          OrderRepository
	model         OrderClassifier // el "modelo de IA"
	fallback      OrderClassifier // reglas deterministas de respaldo
	timeout       time.Duration
	minConfidence float64
}

func NewClassificationService(repo OrderRepository, model, fallback OrderClassifier, timeout time.Duration) *ClassificationService {
	return &ClassificationService{
		repo:          repo,
		model:         model,
		fallback:      fallback,
		timeout:       timeout,
		minConfidence: 0.6, // por debajo, preferimos la regla fija
	}
}

// Classify clasifica un pedido existente.
func (s *ClassificationService) Classify(ctx context.Context, orderID string) (Classification, error) {
	o, err := s.repo.FindByID(ctx, orderID)
	if err != nil {
		return Classification{}, err
	}
	in := toClassificationInput(o)

	c, err := s.askModel(ctx, in)
	if err == nil {
		c.Source = "model"
		return c, nil
	}

	// DEGRADACIÓN CONTROLADA: el modelo falló, tardó o respondió algo inválido.
	// El usuario igual recibe una clasificación, marcada como "rules".
	slog.WarnContext(ctx, "clasificador IA no usable, se aplican reglas", "order_id", orderID, "error", err)
	c, err = s.fallback.Classify(ctx, in)
	if err != nil {
		return Classification{}, fmt.Errorf("clasificar pedido %q: %w", orderID, err)
	}
	c.Source = "rules"
	return c, nil
}

// askModel llama al modelo con un tiempo límite, lo registra (observabilidad)
// y valida su respuesta (evaluación).
func (s *ClassificationService) askModel(ctx context.Context, in ClassificationInput) (Classification, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	start := time.Now()
	c, err := s.model.Classify(ctx, in)
	slog.InfoContext(ctx, "llamada al clasificador IA",
		"duration_ms", time.Since(start).Milliseconds(),
		"label", c.Label, "confidence", c.Confidence, "ok", err == nil)
	if err != nil {
		return Classification{}, err
	}

	switch {
	case !allowedLabels[c.Label]:
		return Classification{}, fmt.Errorf("%w: etiqueta desconocida %q", errInvalidClassification, c.Label)
	case c.Confidence < 0 || c.Confidence > 1:
		return Classification{}, fmt.Errorf("%w: confianza fuera de rango %v", errInvalidClassification, c.Confidence)
	case c.Confidence < s.minConfidence:
		return Classification{}, fmt.Errorf("%w: confianza baja %.2f", errInvalidClassification, c.Confidence)
	}
	return c, nil
}

func toClassificationInput(o *order.Order) ClassificationInput {
	items := o.Items()
	in := ClassificationInput{
		Items:      make([]ClassificationItem, 0, len(items)),
		TotalCents: o.Total().Cents(),
	}
	for _, it := range items {
		in.Items = append(in.Items, ClassificationItem{
			SKU:            it.SKU(),
			Quantity:       it.Quantity(),
			UnitPriceCents: it.UnitPrice().Cents(),
		})
	}
	return in
}
