package httpapi

import (
	"net/http"
	"testing"
)

// Evidencia del PDF: el health check distingue "vivo" de "listo".
func TestHealth_LivenessVsReadiness(t *testing.T) {
	ready := &Readiness{}
	r := NewRouter(Deps{Orders: brokenService{}, Ready: ready})

	steps := []struct {
		name      string
		setReady  bool
		wantLive  int
		wantReady int
	}{
		{name: "arrancando: vivo pero no listo", setReady: false, wantLive: http.StatusOK, wantReady: http.StatusServiceUnavailable},
		{name: "en marcha: vivo y listo", setReady: true, wantLive: http.StatusOK, wantReady: http.StatusOK},
		{name: "apagándose: vivo pero ya no listo", setReady: false, wantLive: http.StatusOK, wantReady: http.StatusServiceUnavailable},
	}

	for _, s := range steps {
		t.Run(s.name, func(t *testing.T) {
			ready.SetReady(s.setReady)

			if got := do(r, http.MethodGet, "/healthz", "").Code; got != s.wantLive {
				t.Errorf("/healthz = %d, se esperaba %d", got, s.wantLive)
			}
			if got := do(r, http.MethodGet, "/readyz", "").Code; got != s.wantReady {
				t.Errorf("/readyz = %d, se esperaba %d", got, s.wantReady)
			}
		})
	}
}
