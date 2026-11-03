package httpapi

import (
	"crypto/rand"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"orders-api/internal/platform/requestid"
)

// HeaderRequestID es el header donde viaja el identificador de correlación.
const HeaderRequestID = requestid.Header

const correlationIDKey = "correlation_id"

// RequestID es un MIDDLEWARE: código que corre antes (y después) de cada
// handler. Asigna un ID único a cada solicitud para poder seguirla en los logs
// y para que el cliente lo reporte si algo falla.
//
// Si el cliente (u otro servicio) ya envía X-Request-ID, lo respetamos; así el
// mismo ID sigue la solicitud a través de varios servicios.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if id == "" || len(id) > 128 { // límite: no confiar ciegamente en lo que manda el cliente
			id = rand.Text()
		}
		c.Set(correlationIDKey, id)
		c.Header(HeaderRequestID, id) // lo devolvemos en la respuesta

		// También en el context.Context de la solicitud: así viaja a la
		// aplicación, al cliente del catálogo y a los logs (J6).
		c.Request = c.Request.WithContext(requestid.With(c.Request.Context(), id))

		c.Next() // continúa con el siguiente middleware o el handler
	}
}

func correlationID(c *gin.Context) string {
	return c.GetString(correlationIDKey)
}

// AccessLog deja UN log estructurado por solicitud, al terminar:
// método, ruta, status y duración. Con esto se pueden calcular las señales
// doradas (tráfico, errores, latencia) aunque todavía no tengamos métricas.
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next() // primero se atiende la solicitud...

		// ...y después se registra, cuando ya sabemos el resultado.
		route := c.FullPath() // "/orders/:id", no "/orders/123": así se agrupan
		if route == "/healthz" || route == "/readyz" {
			return // Kubernetes los llama cada pocos segundos: solo sería ruido
		}
		if route == "" {
			route = "no_route"
		}

		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}

		slog.Log(c.Request.Context(), level, "http request",
			"method", c.Request.Method,
			"route", route,
			"status", status,
			"duration_ms", float64(time.Since(start).Microseconds())/1000,
			"bytes", c.Writer.Size(),
		)
	}
}
