package httpapi

import (
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

// Readiness indica si la API está LISTA para recibir tráfico.
// atomic.Bool: lo escribe main y lo leen las solicitudes (goroutines
// distintas) sin necesidad de mutex.
type Readiness struct {
	ready atomic.Bool
}

func (r *Readiness) SetReady(v bool) { r.ready.Store(v) }
func (r *Readiness) IsReady() bool   { return r.ready.Load() }

// Kubernetes hace dos preguntas distintas a cada pod:
//
//	/healthz (LIVENESS):  "¿estás vivo?"  Si falla varias veces → REINICIA el pod.
//	/readyz  (READINESS): "¿puedes atender?" Si falla → le QUITA el tráfico
//	                      (no lo reinicia) hasta que vuelva a responder bien.
//
// Regla importante: liveness NO revisa dependencias externas. Si el catálogo
// se cae y liveness lo revisara, Kubernetes reiniciaría TODOS los pods de la
// API sin arreglar nada (el problema está en el catálogo). Peor que antes.
//
// Readiness tampoco revisa el catálogo: sin catálogo la API todavía puede
// consultar y confirmar pedidos (falla parcial, J4). Si quitáramos el tráfico,
// convertiríamos una falla parcial en una caída total.

// GET /healthz
func liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GET /readyz
func readiness(r *Readiness) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !r.IsReady() {
			// No listo: arrancando o apagándose (graceful shutdown).
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
