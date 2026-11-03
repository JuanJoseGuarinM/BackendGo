package httpapi

import "github.com/gin-gonic/gin"

// Deps son las dependencias del router. Un struct en vez de muchos
// parámetros: agregar una dependencia nueva no cambia la firma de NewRouter
// ni rompe a quien ya lo usa.
type Deps struct {
	Orders     OrderService
	Classifier Classifier // opcional: sin él no se registra la ruta de clasificación
	Ready      *Readiness
}

// NewRouter arma el servidor Gin con sus middlewares y rutas.
func NewRouter(d Deps) *gin.Engine {
	// gin.New() en vez de gin.Default(): elegimos explícitamente los
	// middlewares, y el ORDEN importa:
	//   1. RequestID: primero, para que todo lo que sigue tenga el ID.
	//   2. AccessLog: envuelve a Recovery, así también registra los panics (500).
	//   3. Recovery: si un handler hace panic, responde 500 sin tumbar el servidor.
	r := gin.New()
	r.Use(RequestID(), AccessLog(), gin.Recovery())

	// Health checks para Kubernetes (J5).
	r.GET("/healthz", liveness)
	r.GET("/readyz", readiness(d.Ready))

	h := &orderHandler{svc: d.Orders}

	orders := r.Group("/orders")
	orders.POST("", h.create)
	orders.GET("/:id", h.get)
	orders.POST("/:id/confirm", h.confirm)
	if d.Classifier != nil {
		orders.GET("/:id/classification", classify(d.Classifier)) // J7
	}

	return r
}
