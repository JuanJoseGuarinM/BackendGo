// Package logging configura los logs ESTRUCTURADOS de los servicios.
//
// Log estructurado = cada línea es un objeto con campos (JSON), no un texto
// libre. Así una herramienta (Loki, Elasticsearch, CloudWatch...) puede
// filtrar y agrupar: "todos los status=503 de la última hora", "todo lo del
// correlation_id=ABC".
//
//	texto:  2026/11/03 INFO pedido creado id=123
//	JSON:   {"time":"...","level":"INFO","msg":"pedido creado","id":"123"}
package logging

import (
	"context"
	"io"
	"log/slog"
	"strings"

	"orders-api/internal/platform/requestid"
)

// New crea el logger del servicio.
//   - format: "json" (por defecto, para producción) o "text" (más legible en local).
//   - level:  "debug", "info" (por defecto), "warn" o "error".
func New(w io.Writer, format, level string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lvl}

	var h slog.Handler
	if strings.EqualFold(format, "text") {
		h = slog.NewTextHandler(w, opts)
	} else {
		h = slog.NewJSONHandler(w, opts)
	}
	return slog.New(contextHandler{Handler: h})
}

// contextHandler envuelve otro handler (composición, J1) y le agrega a cada
// log el correlation_id que viene en el context.
//
// Por eso en el código se usa slog.InfoContext(ctx, ...) en vez de
// slog.Info(...): con el ctx, el log sabe a qué solicitud pertenece sin que
// nadie tenga que escribir el ID a mano.
type contextHandler struct {
	slog.Handler
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := requestid.From(ctx); id != "" {
		r.AddAttrs(slog.String("correlation_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

// WithAttrs y WithGroup deben devolver otro contextHandler; si no, los
// loggers derivados (logger.With(...)) perderían el correlation_id.
func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{Handler: h.Handler.WithGroup(name)}
}
