package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"orders-api/internal/platform/requestid"
)

func TestNew_JSONIncludesCorrelationID(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, "json", "info")
	ctx := requestid.With(t.Context(), "abc-123")

	logger.InfoContext(ctx, "pedido creado", "order_id", "o-1")

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("el log no es JSON válido: %v\n%s", err, buf.String())
	}
	if line["msg"] != "pedido creado" || line["order_id"] != "o-1" || line["correlation_id"] != "abc-123" {
		t.Errorf("log = %v", line)
	}
}

func TestNew_DerivedLoggerKeepsCorrelationID(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, "json", "info").With("service", "orders-api")

	logger.InfoContext(requestid.With(t.Context(), "abc-123"), "hola")

	if !strings.Contains(buf.String(), `"correlation_id":"abc-123"`) {
		t.Errorf("el logger derivado perdió el correlation_id: %s", buf.String())
	}
}

func TestNew_LevelFiltersDebug(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, "json", "info")

	logger.Debug("detalle interno")

	if buf.Len() != 0 {
		t.Errorf("con nivel info no debía salir el debug: %s", buf.String())
	}
}
