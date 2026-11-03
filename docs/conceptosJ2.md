# Guía de conceptos — Jornada 2: API con Gin y contratos de respuesta

Cada concepto tiene: **qué es**, **dónde está en el proyecto** y **cómo explicarlo en una entrevista**.

Correr la API: `go run ./cmd/api` · Pruebas: `go test ./internal/...` · Decisiones: [decisiones.md](decisiones.md)

---

### Arquitectura en capas (hexagonal)
- **Qué es:** el código se separa por responsabilidad: **dominio** (reglas), **aplicación** (casos de uso), **adaptadores** (HTTP, repositorio). Las dependencias apuntan hacia adentro.
- **En el proyecto:** [domain/order](../internal/domain/order/order.go) ← [application](../internal/application/order_service.go) ← [adapters/httpapi](../internal/adapters/httpapi/handlers.go) y [adapters/repository/memory](../internal/adapters/repository/memory/order_repository.go). Las arma [cmd/api/main.go](../cmd/api/main.go).
- **En entrevista:** "El dominio no conoce HTTP ni la base de datos. Así las reglas se prueban solas y puedo cambiar Gin o la persistencia sin tocarlas."

### HTTP: recursos y métodos
- **Recurso:** una "cosa" con URL: `/orders`, `/orders/123`.
- **Métodos:** `GET` consulta (no cambia nada), `POST` crea o ejecuta una acción, `PUT` reemplaza, `PATCH` modifica parcialmente, `DELETE` borra.
- **En el proyecto:** `POST /orders` (crear), `GET /orders/:id` (consultar), `POST /orders/:id/confirm` (acción) en [router.go](../internal/adapters/httpapi/router.go).
- **En entrevista:** "Confirmar lo modelé como `POST /orders/:id/confirm` porque es una acción del negocio con reglas, no editar un campo; así nadie manda `status: CONFIRMED` saltándose las validaciones."

### Idempotencia
- **Qué es:** una operación es idempotente si hacerla **1 vez o N veces deja el mismo resultado**. `GET`, `PUT` y `DELETE` deben serlo; `POST` normalmente no (dos POST = dos pedidos).
- **Por qué importa:** si una solicitud falla por red, el cliente **reintenta**. Reintentar algo no idempotente puede duplicar (dos cobros, dos pedidos).
- **En el proyecto:** confirmar dos veces no cambia nada (la segunda responde 409); el pedido queda confirmado una sola vez.
- **En entrevista:** "Solo reintento automáticamente operaciones idempotentes. Para hacer un POST idempotente se usa una *idempotency key* que el servidor recuerda."

### Códigos de estado HTTP
- **2xx éxito:** 200 OK, 201 Created (+ header `Location`).
- **4xx error del cliente:** 400 request mal formado, 404 no existe, 409 conflicto con el estado actual, 422 bien formado pero viola una regla.
- **5xx error del servidor:** 500 falla inesperada.
- **En el proyecto:** [errors.go](../internal/adapters/httpapi/errors.go), el único lugar donde se decide el código.
- **En entrevista:** "4xx es culpa del cliente y no sirve reintentar igual; 5xx es culpa del servidor y puede reintentarse. Distingo 400 (mal escrito) de 422 (viola una regla del negocio)."

### Entrada mal formada vs violación de regla del negocio
- **Mal formada (400):** JSON roto, falta `customer_id`, `quantity` en 0. Lo detecta el **adaptador HTTP** con `ShouldBindJSON` + etiquetas `binding`.
- **Regla del negocio (409/422):** confirmar un pedido vacío o ya confirmado. Lo detecta el **dominio** en `Confirm()`.
- **En entrevista:** "La forma del JSON se valida en el borde HTTP; las reglas del negocio en el dominio. Son dos validaciones distintas en dos capas distintas."

### DTO (Data Transfer Object)
- **Qué es:** structs que representan el JSON que **entra** y **sale**. Son distintos de los tipos del dominio.
- **En el proyecto:** `createOrderRequest`, `orderResponse`, `errorResponse` en [dto.go](../internal/adapters/httpapi/dto.go).
- **En entrevista:** "Separo DTOs del dominio para que el contrato público de la API no cambie si cambio el modelo interno, y para no ensuciar el dominio con etiquetas `json` o `binding`."

### `ShouldBindJSON` y validación estructural
- **Qué es:** lee el JSON del body, lo pasa al struct y aplica las reglas de las etiquetas `binding:"required,gt=0"`.
- **`ShouldBindJSON` vs `BindJSON`:** `Bind*` responde 400 automáticamente con su propio formato; `ShouldBind*` solo devuelve el error y **yo** decido la respuesta.
- **En el proyecto:** `create` en [handlers.go](../internal/adapters/httpapi/handlers.go).

### Middleware
- **Qué es:** código que se ejecuta **antes y/o después de cada handler**: logs, autenticación, request ID, recuperación de panics. `c.Next()` pasa al siguiente.
- **En el proyecto:** `RequestID()` en [middleware.go](../internal/adapters/httpapi/middleware.go) y `gin.Recovery()` en [router.go](../internal/adapters/httpapi/router.go).
- **En entrevista:** "Uso middlewares para lo transversal, lo que aplica a todas las rutas, así no lo repito en cada handler."

### Correlation ID (Request ID)
- **Qué es:** un identificador único por solicitud. Viaja en el header `X-Request-ID`, va en la respuesta de error y en los logs.
- **En el proyecto:** [middleware.go](../internal/adapters/httpapi/middleware.go); se incluye en `errorResponse` y en el log de los 500 en [errors.go](../internal/adapters/httpapi/errors.go).
- **En entrevista:** "Si un cliente reporta un error, con el correlation ID encuentro exactamente esa solicitud en los logs, incluso si pasó por varios servicios."

### No exponer detalles internos
- **Qué es:** un 500 nunca muestra al cliente el error real (IPs, SQL, rutas), porque eso ayuda a un atacante. El detalle va al **log**.
- **En el proyecto:** caso `default` en `writeError` ([errors.go](../internal/adapters/httpapi/errors.go)); lo prueba `TestUnexpectedError_DoesNotLeakDetails` en [handlers_test.go](../internal/adapters/httpapi/handlers_test.go).

### `gin.Context` vs `context.Context`
- **`*gin.Context`:** objeto de **Gin** para la solicitud HTTP actual: leer parámetros y body, escribir la respuesta. Solo vive en el adaptador HTTP.
- **`context.Context`:** de la librería estándar: **cancelación y deadline**. Se obtiene con `c.Request.Context()` y se pasa a las capas internas.
- **En el proyecto:** cada handler hace `h.svc.Create(c.Request.Context(), ...)` en [handlers.go](../internal/adapters/httpapi/handlers.go).
- **En entrevista:** "A la capa de aplicación le paso `c.Request.Context()`, nunca el `gin.Context`, para no acoplar el negocio a Gin. Si el cliente se desconecta, ese context se cancela y el trabajo se detiene."

### Gin es un adaptador, no el lugar de las reglas
- **En entrevista:** "Gin solo traduce HTTP ↔ casos de uso: valida la forma, llama al servicio y traduce el resultado. Si mañana expongo lo mismo por gRPC o por una cola, reutilizo el mismo dominio y la misma aplicación; solo cambia el adaptador."

### Pruebas de handlers con `httptest`
- **Qué es:** `httptest.NewRequest` + `httptest.NewRecorder` permiten probar la API **sin levantar un servidor** ni usar la red.
- **En el proyecto:** [handlers_test.go](../internal/adapters/httpapi/handlers_test.go): éxito, entrada inválida (400), regla de negocio (422/409), no encontrado (404), error inesperado (500) y request ID.

### Autenticación y JWT (para sustentar; no implementado)
- **Autenticación:** *¿quién eres?* **Autorización:** *¿qué puedes hacer?*
- **JWT:** un token firmado que el cliente envía en `Authorization: Bearer <token>`. Contiene datos (*claims*: usuario, roles, expiración). El servidor verifica la **firma** sin consultar una base de datos.
- **Dónde va:** en un **middleware** que valida el token antes del handler y responde 401 (no autenticado) o 403 (sin permiso).
- **Cuidados:** verificar firma y expiración, no guardar datos sensibles (el contenido se puede leer; solo está firmado), tiempos de vida cortos, secretos fuera del código.
- **En entrevista:** "La autenticación es transversal, así que va en un middleware; el dominio recibe, como mucho, el ID del usuario ya validado."

### Apagado controlado (graceful shutdown)
- **Qué es:** al recibir Ctrl+C o `SIGTERM`, el servidor deja de aceptar solicitudes y termina las que tiene en curso antes de salir.
- **En el proyecto:** `run()` en [cmd/api/main.go](../cmd/api/main.go).
- **En entrevista:** "Kubernetes envía SIGTERM antes de matar un pod; con graceful shutdown no corto solicitudes a la mitad durante un despliegue."
