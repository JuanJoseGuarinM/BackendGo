package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"orders-api/internal/platform/logging"
)

// captureLogs reemplaza el logger global por uno que escribe en un buffer
// durante la prueba, y lo restaura al final.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(logging.New(&buf, "json", "debug"))
	t.Cleanup(func() { slog.SetDefault(old) })
	return &buf
}

// Evidencia del PDF: una solicitud deja un log estructurado interpretable.
func TestAccessLog_OneStructuredLinePerRequest(t *testing.T) {
	logs := captureLogs(t)
	r := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/orders/no-existe", nil)
	req.Header.Set(HeaderRequestID, "req-42")
	r.ServeHTTP(httptest.NewRecorder(), req)

	var line map[string]any
	if err := json.Unmarshal(logs.Bytes(), &line); err != nil {
		t.Fatalf("se esperaba UNA línea JSON: %v\n%s", err, logs.String())
	}
	want := map[string]any{
		"msg":            "http request",
		"level":          "WARN", // 404 → warn
		"method":         "GET",
		"route":          "/orders/:id", // la plantilla, no el ID: así se agrupa
		"status":         float64(404),
		"correlation_id": "req-42",
	}
	for k, v := range want {
		if line[k] != v {
			t.Errorf("%s = %v, se esperaba %v", k, line[k], v)
		}
	}
	if _, ok := line["duration_ms"]; !ok {
		t.Error("falta duration_ms")
	}
}

func TestAccessLog_SkipsHealthChecks(t *testing.T) {
	logs := captureLogs(t)
	r := newTestRouter()

	do(r, http.MethodGet, "/healthz", "")
	do(r, http.MethodGet, "/readyz", "")

	if logs.Len() != 0 {
		t.Errorf("los health checks no debían registrarse: %s", logs.String())
	}
}

func TestErrorLog_HasCorrelationIDFromContext(t *testing.T) {
	logs := captureLogs(t)
	r := newTestRouterWithCatalog(downCatalog{})

	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(validOrder))
	req.Header.Set(HeaderRequestID, "req-99")
	r.ServeHTTP(httptest.NewRecorder(), req)

	// Dos líneas: el aviso del catálogo y el access log. Ambas con el mismo ID.
	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("líneas = %d, se esperaban 2\n%s", len(lines), logs.String())
	}
	for _, l := range lines {
		if !strings.Contains(l, `"correlation_id":"req-99"`) {
			t.Errorf("línea sin correlation_id: %s", l)
		}
	}
}
