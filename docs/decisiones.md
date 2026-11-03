# Decisiones de arquitectura

Registro breve de las decisiones del proyecto: qué se decidió, por qué y qué límites tiene.

---

## D1 — Arquitectura hexagonal (puertos y adaptadores)
- **Decisión:** separar `domain` (reglas), `application` (casos de uso) y `adapters` (HTTP, repositorio, catálogo).
- **Por qué:** las reglas del negocio se prueban sin Gin ni base de datos, y se puede cambiar un adaptador sin tocar el dominio.
- **Regla:** los `import` apuntan hacia adentro. `domain` no importa a nadie del proyecto.

## D2 — Interfaces definidas por el consumidor
- **Decisión:** `application.OrderRepository` vive en `application` (quien la usa), y `httpapi.OrderService` vive en `httpapi`.
- **Por qué:** cada capa declara solo lo que necesita; las implementaciones no dependen de quien las usa.

## D3 — Códigos HTTP de error
| Error | HTTP | Código |
|---|---|---|
| JSON mal formado o no cumple la forma (`binding`) | 400 | `INVALID_REQUEST` |
| Pedido no existe | 404 | `ORDER_NOT_FOUND` |
| Pedido ya confirmado | 409 | `ORDER_ALREADY_CONFIRMED` |
| Confirmar pedido vacío | 422 | `ORDER_EMPTY` |
| Datos que violan una regla del dominio | 422 | `INVALID_ORDER` |
| Cualquier otro | 500 | `INTERNAL_ERROR` |

- **400 vs 422:** 400 = el request está *mal escrito* (problema de transporte). 422 = está bien escrito pero *viola una regla del negocio*.
- **409 vs 422 para el pedido vacío:** el PDF sugiere 409 para conflictos de estado y su ejemplo usa 422 para el pedido vacío. Elegimos:
  - **409** cuando el problema es el **estado actual** del recurso (ya está confirmado: reintentar no tiene sentido).
  - **422** cuando el **contenido** del pedido no cumple la regla (no tiene productos).
- Los 500 nunca exponen detalles internos al cliente; el detalle va al log con el `correlation_id`.

## D4 — Formato único de error con correlation ID
- **Decisión:** todos los errores responden `{code, message, correlation_id}`. El ID viaja en el header `X-Request-ID`; si el cliente lo envía, se respeta.
- **Por qué:** el cliente programa contra `code` (estable), no contra el texto. Con el `correlation_id` se encuentra la solicitud en los logs.

## D5 — Dinero en centavos (`int64`) con Value Object `Money`
- **Por qué:** `float64` tiene errores de redondeo (0.1 + 0.2 ≠ 0.3).
- **J3:** se encapsuló en el Value Object `Money` (inmutable, nunca negativo). Es el único value object del modelo porque es el único con una invariante clara; `SKU` y `CustomerID` siguen siendo `string`.

## D6 — Repositorio en memoria
- **Por qué:** suficiente para el ejercicio y para pruebas; se reemplaza sin tocar dominio ni casos de uso.
- **Límite conocido:** si llegan dos `confirm` del mismo pedido **al mismo tiempo**, ambos podrían leer `PENDING` y ambos responder 200 (*lost update*). Con una base de datos se resuelve con una transacción o con bloqueo optimista (columna `version`). Tema de la J3.

## D7 — `POST /orders/:id/confirm` en vez de `PUT`/`PATCH`
- **Por qué:** confirmar es una **acción** del negocio con reglas, no editar un campo. Así el cliente no puede mandar `"status": "CONFIRMED"` saltándose las reglas.
- **Idempotencia:** no es idempotente en la respuesta (la segunda vez da 409), pero sí en el efecto: el pedido queda confirmado una sola vez.

## D9 — `Order` como agregado con campos privados (J3)
- **Decisión:** `Order` (raíz) + `OrderItem` forman un agregado. Todos los campos son privados; el estado solo cambia con `Confirm` y `AddItem`.
- **Por qué:** en la J2 cualquiera podía escribir `o.Status = StatusConfirmed` y saltarse la regla del pedido vacío. Ahora el compilador lo impide.
- **Costo:** más código (getters, `Restore`, traducción en el repositorio). Se acepta porque protege la regla principal del negocio.

## D10 — El repositorio traduce a un formato propio (`orderRecord`)
- **Decisión:** el repositorio no guarda el objeto del dominio tal cual; lo convierte a datos planos y al leer lo reconstruye con `Restore`.
- **Por qué:** es lo que pasará con PostgreSQL (filas y columnas). El dominio no sabe cómo se guarda.
- **Detalle:** si un dato guardado es inválido, el error se envuelve con `%v` (no `%w`) para que la API responda 500 y no 422: es falla nuestra, no del cliente.

## D11 — El precio lo pone el catálogo, no el cliente (J4)
- **Decisión:** `POST /orders` ya no acepta `unit_price`; el caso de uso consulta el precio con `CatalogClient`.
- **Por qué:** si el cliente envía el precio, puede comprar a $0. Los datos sensibles los decide el servidor.
- **Costo:** crear un pedido depende de que el catálogo responda (acoplamiento síncrono).

## D12 — HTTP/REST para el catálogo, y cuándo usaría otra cosa (J4)
- **Elegimos HTTP + JSON** porque: el contrato es pequeño (un GET), es fácil de probar (`httptest`, curl), no necesita generar código y cualquier equipo lo entiende.
- **gRPC** sería razonable si el catálogo fuera un servicio interno con **mucho tráfico** y baja latencia, o con contratos estrictos versionados (`.proto`).
- **SOAP** sería razonable si el catálogo fuera un **sistema legado** (ERP, banco) que ya expone un WSDL.
- **Mensajería asíncrona** (cola / pub-sub) sería razonable cuando **no necesito la respuesta para continuar**: notificar al cliente, actualizar inventario o analítica después de confirmar un pedido. No sirve para el precio, porque sin precio no hay pedido.

## D13 — Política de resiliencia hacia el catálogo (J4)
- **Timeout por intento:** 1 s. **Reintentos:** 2 (3 intentos), con backoff exponencial + jitter.
- **Se reintenta:** errores de red, timeouts, 5xx y 429. **No se reintenta:** 404 (respuesta definitiva) ni respuestas inválidas.
- **Por qué se puede reintentar:** `GET /products/{sku}` es seguro e idempotente.
- **Si se agota:** `ErrCatalogUnavailable` → la API responde **503** `CATALOG_UNAVAILABLE`.
- **Peor caso:** ~3,4 s (3 × 1 s + esperas). Si fuera mucho, se baja el timeout o los reintentos.
- **Pendiente:** circuit breaker si el catálogo fallara seguido.

## D14 — La API corre con 1 réplica en Kubernetes (J5)
- **Decisión:** `replicas: 1` para `orders-api`; `replicas: 2` para `catalog`.
- **Por qué:** los pedidos viven en la memoria del pod. Con 2 réplicas, un pedido creado en un pod no existiría en el otro, y el Service reparte el tráfico entre ambos. El catálogo no tiene estado, así que sí escala.
- **Para escalar (y usar HPA):** mover los pedidos a PostgreSQL. El cambio es solo un nuevo adaptador de `OrderRepository` (hexagonal).

## D15 — Liveness y readiness no revisan el catálogo (J5)
- **Liveness (`/healthz`):** solo "el proceso responde". Si revisara el catálogo, una caída del catálogo haría que Kubernetes reiniciara todos los pods de la API sin arreglar nada (reinicios en cascada).
- **Readiness (`/readyz`):** "listo para atender". Responde 503 al arrancar y durante el apagado (drenado con `SHUTDOWN_DELAY`). No revisa el catálogo porque sin él la API todavía puede consultar y confirmar pedidos (falla parcial).
- **Si hubiera PostgreSQL:** readiness sí podría revisarlo, porque sin BD la API no puede hacer nada.

## D16 — Imagen distroless sin root (J5)
- **Decisión:** build multistage → `gcr.io/distroless/static-debian12:nonroot`, `USER 65532`, `readOnlyRootFilesystem` y `capabilities: drop ALL` en Kubernetes.
- **Por qué:** imagen pequeña (~36 MB), sin shell ni herramientas que un atacante pueda usar, y sin privilegios.
- **Costo:** no se puede hacer `docker exec ... sh` para depurar dentro del contenedor.

## D17 — Logs JSON con correlation ID en el context (J6)
- **Decisión:** `log/slog` con salida JSON (`LOG_FORMAT=text` para leer en local). El ID de la solicitud viaja en el `context.Context` (`internal/platform/requestid`) y un handler de slog lo agrega solo a cada log que use `*Context(ctx, ...)`.
- **Por qué:** nadie tiene que acordarse de escribir el ID a mano, y el mismo ID se envía al catálogo (`X-Request-ID`), así que una solicitud se sigue entre servicios.
- **Un access log por solicitud**, nivel según el status (5xx = ERROR, 4xx = WARN). Se omiten `/healthz` y `/readyz` para no llenar los logs de ruido.

## D18 — Pipeline de CI en tres jobs paralelos (J6)
- **quality:** gofmt, go vet, golangci-lint (incluye gosec), pruebas con `-race` y cobertura.
- **security:** govulncheck (dependencias).
- **image:** build + Trivy (falla con HIGH/CRITICAL que tengan arreglo). Solo corre si `quality` pasó.
- **Local:** `scripts/verify.ps1 -Race` repite lo esencial; `-race` corre dentro de Docker porque en Windows necesitaría gcc.
- **Límite:** el CI todavía no se ha ejecutado porque el repositorio no está en GitHub.
- **Ejecutado localmente con Docker (2026-11-03):**
  - `golangci-lint` (latest) → **0 issues**. La v2.5.0 no sirve: está compilada con Go 1.25 y el proyecto usa 1.27.
  - `govulncheck` → primero encontró **5 vulnerabilidades** en dependencias **indirectas** (traídas por Gin): `golang.org/x/net` v0.51.0 (4, HTTP/2) y `quic-go` v0.59.0 (1, HTTP/3). Se actualizaron a v0.60.0 y v0.59.1; las pruebas siguieron pasando y el nuevo escaneo dio **0 vulnerabilidades**.
  - `Trivy` sobre `orders-api:dev` → **0 HIGH/CRITICAL** (sistema base Debian 12 y binario Go).
- **Primer CI en GitHub (2026-11-03):** calidad y pruebas pasaron; govulncheck falló porque el CI instaló Go **1.27.0** (de `go 1.27.0` en go.mod), que tiene vulnerabilidades en la librería estándar corregidas en **1.27.3** (localmente se escaneó en el contenedor `golang:1.27`, que ya era 1.27.3). Se fijó `toolchain go1.27.3` en go.mod: CI y máquinas locales usan la misma versión exacta. También se corrigió la versión de la acción de Trivy (`v0.36.0`).

## D19 — Terraform con el provider de Docker (J6)
- **Por qué Docker y no AWS:** practicar el ciclo completo (plan, apply, state, destroy) sin una cuenta en la nube ni costos. Los conceptos son los mismos.
- **State local e ignorado en Git;** en equipo iría en un backend remoto con bloqueo.

## D20 — IA aislada, opcional y con respaldo (J7)
- **Decisión:** la clasificación va en `GET /orders/:id/classification`, separada de crear y confirmar. El caso de uso llama al modelo con **timeout de 1 s**, **valida** la respuesta (etiqueta permitida, confianza entre 0 y 1, mínimo 0.6) y, si algo falla, usa **reglas deterministas**. La respuesta indica `source: model | rules`.
- **Por qué:** un modelo puede fallar, tardar, alucinar o dudar. Ninguna operación del negocio debe depender de él, y la API nunca responde 5xx por su culpa.
- **Privacidad:** `ClassificationInput` no tiene el cliente (minimización de datos): es imposible enviarlo al proveedor por error.
- **Simulado a propósito:** el PDF pide no contratar un proveedor. `CLASSIFIER_MODE` permite reproducir cada riesgo (`fail`, `invalid`, `slow`).

## D21 — `httpapi.Deps` en vez de parámetros sueltos (J7)
- **Decisión:** `NewRouter(Deps{Orders, Classifier, Ready})`.
- **Por qué:** con cada jornada el router recibía una dependencia más. Un struct evita cambiar la firma (y todos los llamados) cada vez; las dependencias opcionales simplemente se omiten.

## D8 — Apagado controlado (graceful shutdown)
- **Decisión:** al recibir Ctrl+C o `SIGTERM`, el servidor deja de aceptar solicitudes y termina las que están en curso (máx. 10 s).
- **Por qué:** Kubernetes envía `SIGTERM` antes de matar un pod; así no se cortan solicitudes a la mitad (J5).
