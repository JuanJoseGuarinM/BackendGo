# orders-api

API REST de **gestión de pedidos** en **Go + Gin**, con arquitectura hexagonal y DDD, integración resiliente con un catálogo externo, clasificación de pedidos con IA (simulada) y lista para correr en Docker y Kubernetes.

> Proyecto de práctica de una ruta de fortalecimiento backend. Todos los datos son ficticios.

---

## Qué hace

- **Crear, consultar y confirmar pedidos.** Regla principal: un pedido **sin productos no se puede confirmar**.
- **El precio lo pone el catálogo**, nunca el cliente. La API consulta un servicio de catálogo con timeout, reintentos y backoff.
- **Clasificación con IA**: etiqueta cada pedido como `STANDARD`, `BULK` o `REVIEW`. Si el modelo falla, tarda o responde algo inválido, se usan reglas fijas: la IA **nunca** tumba la API.
- **Lista para operar**: health checks, apagado controlado, logs JSON con correlation ID entre servicios, imagen distroless sin root, manifiestos de Kubernetes y CI con escaneo de seguridad.

## Arquitectura

```
                ┌────────────────── adapters ──────────────────┐
  HTTP (Gin) ──►│ httpapi        repository/memory    catalog ─┼──► servicio de catálogo (HTTP)
                │    │                  ▲             classifier│──► modelo de IA (simulado)
                └────┼──────────────────┼──────────────▲────────┘
                     ▼                  │              │
                ┌──────────── application (casos de uso) ───────┐
                │ OrderService · ClassificationService          │
                │ puertos: OrderRepository, CatalogClient,      │
                │          OrderClassifier                      │
                └───────────────────────┬───────────────────────┘
                                        ▼
                ┌──────────────── domain/order ─────────────────┐
                │ Order (agregado) · OrderItem · Money (VO)      │
                │ reglas: no confirmar vacío, no cambiar si está │
                │ confirmado, montos nunca negativos             │
                └───────────────────────────────────────────────┘
```

- Las dependencias **apuntan hacia adentro**: el dominio solo usa la librería estándar de Go.
- Las **interfaces (puertos) las define quien las usa** (`application`); los adaptadores las implementan.
- `cmd/api/main.go` arma las piezas (inyección de dependencias).

## Estructura

```
cmd/
  api/            punto de entrada de la API
  catalog/        catálogo ficticio (segundo servicio)
  practice/       demo de los ejercicios de fundamentos de Go
internal/
  domain/order/   reglas del negocio (sin dependencias externas)
  application/    casos de uso y puertos
  adapters/
    httpapi/      Gin: handlers, DTOs, errores → HTTP, middlewares, health
    catalog/      cliente HTTP resiliente + catálogo en memoria
    classifier/   modelo de IA simulado + reglas de respaldo
    repository/   persistencia en memoria
  platform/       logging estructurado y request ID
  practice/       ejercicios de errores, interfaces y concurrencia
deploy/
  k8s/            Deployment, Service, ConfigMap (kustomize)
  terraform/      ejemplo de IaC con el provider de Docker
docs/             decisiones, observabilidad, demo, guías de estudio
```

---

## Inicio rápido

**Requisitos:** Go 1.27. Para contenedores: Docker (y `kind` para Kubernetes).

```powershell
go run ./cmd/api          # API en http://localhost:8080 (catálogo en memoria)
```

```powershell
# Crear un pedido
$p = Invoke-RestMethod -Method Post -Uri http://localhost:8080/orders -ContentType "application/json" `
     -Body '{"customer_id":"c1","items":[{"sku":"SKU-1","quantity":2}]}'

Invoke-RestMethod http://localhost:8080/orders/$($p.id)                       # consultar
Invoke-RestMethod http://localhost:8080/orders/$($p.id)/classification        # clasificar (IA)
Invoke-RestMethod -Method Post -Uri http://localhost:8080/orders/$($p.id)/confirm   # confirmar
```

Productos de ejemplo: `SKU-1` Café ($15.00), `SKU-2` Arepas ($8.00), `SKU-3` Chocolatina ($5.00). Los montos van en **centavos**.

**Con el catálogo como servicio aparte** (para ver reintentos y fallas):
```powershell
go run ./cmd/catalog                                           # terminal 1 → :8081
$env:CATALOG_URL="http://localhost:8081"; go run ./cmd/api      # terminal 2
```

## Endpoints

| Método | Ruta | Descripción | Éxito |
|---|---|---|---|
| `POST` | `/orders` | Crea un pedido. Body: `{"customer_id","items":[{"sku","quantity"}]}` | `201` + `Location` |
| `GET` | `/orders/:id` | Consulta un pedido | `200` |
| `POST` | `/orders/:id/confirm` | Confirma un pedido | `200` |
| `GET` | `/orders/:id/classification` | Clasifica el pedido (IA con respaldo de reglas) | `200` |
| `GET` | `/healthz` | Liveness: el proceso está vivo | `200` |
| `GET` | `/readyz` | Readiness: listo para recibir tráfico | `200` / `503` |

**Errores:** siempre con el mismo formato `{"code","message","correlation_id"}` y sin detalles internos.

| HTTP | `code` | Cuándo |
|---|---|---|
| 400 | `INVALID_REQUEST` | JSON mal formado o campos faltantes |
| 404 | `ORDER_NOT_FOUND` | El pedido no existe |
| 409 | `ORDER_ALREADY_CONFIRMED` | El pedido ya estaba confirmado |
| 422 | `ORDER_EMPTY` | Confirmar un pedido sin productos |
| 422 | `PRODUCT_NOT_FOUND` | Un SKU no existe en el catálogo |
| 422 | `INVALID_ORDER` | Datos que violan una regla del dominio |
| 503 | `CATALOG_UNAVAILABLE` | El catálogo no responde (tras los reintentos) |
| 500 | `INTERNAL_ERROR` | Falla inesperada (el detalle queda en el log) |

## Configuración

| Variable | Para qué | Por defecto |
|---|---|---|
| `PORT` | Puerto del servicio | `8080` (catálogo: `8081`) |
| `CATALOG_URL` | URL del catálogo; vacío = catálogo en memoria | vacío |
| `CLASSIFIER_MODE` | Simula riesgos de la IA: `fail`, `invalid`, `slow` | vacío (ok) |
| `LOG_FORMAT` | `json` o `text` | `json` |
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` | `info` |
| `SHUTDOWN_DELAY` | Tiempo de drenado al apagarse (ej. `5s`) | `0` |
| `CATALOG_DELAY` | (solo `cmd/catalog`) simula lentitud | `0` |

---

## Calidad y seguridad

```powershell
go test ./...
.\scripts\verify.ps1 -Race      # gofmt + go vet + pruebas con cobertura + -race (en Docker)
```

El CI ([.github/workflows/ci.yml](.github/workflows/ci.yml)) corre en cada push: formato, `go vet`, **golangci-lint** (con **gosec**), pruebas con `-race` y cobertura, **govulncheck** y escaneo de la imagen con **Trivy**.
Última ejecución local de las herramientas de seguridad: golangci-lint **0 issues**, govulncheck **0 vulnerabilidades** (tras actualizar dependencias indirectas), Trivy **0 HIGH/CRITICAL**.

## Docker

```powershell
docker build -t orders-api:dev .
docker build -t catalog:dev --build-arg TARGET=catalog .
docker run --rm -p 8080:8080 orders-api:dev
```
Imagen multistage → `distroless/static`, ~36 MB, usuario sin privilegios (65532).

## Kubernetes (kind)

```powershell
kind create cluster --name orders
kind load docker-image orders-api:dev catalog:dev --name orders
kubectl apply -k deploy/k8s
kubectl get pods -n orders
kubectl port-forward -n orders service/orders-api 8080:80
```
Rolling update sin caída, probes de liveness/readiness, requests/limits y contexto de seguridad restrictivo. La API corre con **1 réplica** porque guarda los pedidos en memoria (ver *Limitaciones*).

## Infraestructura como código

[deploy/terraform](deploy/terraform/) levanta la API y el catálogo con el provider de Docker (ciclo `init → plan → apply → destroy`, Terraform corriendo dentro de Docker). Comandos en [README2.md](README2.md) (Jornada 6, paso 3) y conceptos en [docs/observabilidad.md](docs/observabilidad.md).

## Observabilidad

- Logs **JSON** con `log/slog`; un log por solicitud (`route`, `status`, `duration_ms`).
- **Correlation ID** (`X-Request-ID`) en cada log, en las respuestas de error y **propagado al catálogo**.
- Qué medir (4 señales doradas) y dónde instrumentar con OpenTelemetry: [docs/observabilidad.md](docs/observabilidad.md).

---

## Decisiones de diseño

Detalle de cada decisión en [docs/decisiones.md](docs/decisiones.md). Las principales:

- **Hexagonal + DDD:** `Order` es un agregado con campos privados; sus invariantes solo cambian vía `Confirm`/`AddItem`. `Money` es un value object que nunca es negativo.
- **409 vs 422:** 409 cuando el problema es el *estado* del recurso (ya confirmado); 422 cuando el *contenido* viola una regla (pedido vacío).
- **¿Por qué HTTP para el catálogo?** Contrato pequeño y fácil de probar. **gRPC** para servicios internos de alto tráfico; **SOAP** para sistemas legados con WSDL; **mensajería asíncrona** para lo que no necesita respuesta inmediata (notificaciones, inventario), no para el precio.
- **Resiliencia:** timeout de 1 s por intento, 2 reintentos con backoff exponencial + jitter, solo en el GET (idempotente). Catálogo caído → 503 controlado.
- **IA aislada:** detrás de `OrderClassifier`, en un endpoint aparte, con timeout, validación de la respuesta, umbral de confianza y reglas de respaldo. Al modelo solo se envían los datos necesarios (sin el cliente).
- **Health checks:** liveness no revisa dependencias (evita reinicios en cascada); readiness pasa a 503 durante el apagado.

## Limitaciones y cómo evolucionaría

| Limitación hoy | Siguiente paso |
|---|---|
| Pedidos en memoria → 1 réplica, se pierden al reiniciar | PostgreSQL como nuevo adaptador de `OrderRepository` + bloqueo optimista (`version`) para confirmaciones simultáneas; luego HPA |
| Sin autenticación | Middleware JWT (401/403); el dominio recibe solo el ID del usuario |
| IA simulada | Adaptador a un modelo aprobado por HTTP, API key en un gestor de secretos, evaluación con un set de casos conocidos |
| Sin métricas ni trazas | OpenTelemetry (`otelgin`, `otelhttp`) → Collector → Prometheus / Tempo / Grafana |
| Catálogo consultado producto por producto | Deduplicar SKUs o consultar en paralelo; circuit breaker si falla seguido |
| CI definido pero aún no ejecutado en GitHub | Subir el repositorio y fijar la versión del linter |

## Documentación

| Documento | Contenido |
|---|---|
| [docs/demo.md](docs/demo.md) | Guion de la demo de 10 minutos |
| [docs/decisiones.md](docs/decisiones.md) | Registro de decisiones de arquitectura |
| [docs/observabilidad.md](docs/observabilidad.md) | IaC, métricas, OpenTelemetry, diagnóstico de latencia |
| [README2.md](README2.md) | Orden de estudio del código, jornada por jornada |
| `docs/conceptosJ1.md` … `conceptosJ7.md` | Guías de conceptos para la sustentación |
