# Orden de estudio — Jornada 1

Lee los archivos en este orden. Cada uno se apoya en el anterior.
Por cada paso: **lee el código → lee su prueba → corre el comando → repasa el concepto en [docs/conceptosJ1.md](docs/conceptosJ1.md).**

---

## Paso 0 — Estructura del proyecto
| Archivo | Qué mirar |
|---|---|
| [go.mod](go.mod) | El nombre del módulo (`orders-api`) y la versión de Go. |
| [cmd/api/main.go](cmd/api/main.go) | Un `package main` con `func main()`: así arranca un programa. |

**Concepto:** módulo, paquete, `cmd/` e `internal/`.

---

## Paso 1 — Errores
| Archivo | Qué mirar |
|---|---|
| [internal/practice/inventory/inventory.go](internal/practice/inventory/inventory.go) | `ErrProductNotFound` (error con nombre), `ValidationError` (error con datos), `%w` (envolver). |
| [internal/practice/inventory/inventory_test.go](internal/practice/inventory/inventory_test.go) | La **tabla** de casos y cómo se usan `errors.Is` y `errors.As`. |
| [cmd/practice/errors.go](cmd/practice/errors.go) | Cómo se decide qué hacer según el error (anticipo de los códigos HTTP). |

```powershell
go test -v ./internal/practice/inventory/
```

**Conceptos:** errores como valores, centinela, `%w`, `errors.Is` / `errors.As`, error vs panic, prueba table-driven.

---

## Paso 2 — Interfaces
| Archivo | Qué mirar |
|---|---|
| [internal/practice/checkout/checkout.go](internal/practice/checkout/checkout.go) | La interfaz `Notifier` está definida **aquí**, donde se usa. `NewService` recibe la dependencia desde afuera. |
| [internal/practice/checkout/checkout_test.go](internal/practice/checkout/checkout_test.go) | `failingNotifier`: un falso para simular un fallo. Las líneas `var _ Notifier = ...` comprueban en compilación. |

```powershell
go test -v ./internal/practice/checkout/
```

**Conceptos:** interfaces implícitas, criterios para definir una interfaz, inyección de dependencias, doble de prueba.

---

## Paso 3 — Receptores, composición y mutex
| Archivo | Qué mirar |
|---|---|
| [internal/practice/notify/notify.go](internal/practice/notify/notify.go) | `ConsoleNotifier` (receptor **por valor**) vs `MemoryNotifier` (receptor **por puntero** + mutex). `PrefixNotifier` (composición). |
| [internal/practice/notify/notify_test.go](internal/practice/notify/notify_test.go) | Prueba con 100 goroutines escribiendo a la vez. `p.Sent()` funcionando gracias al embedding. |
| [cmd/practice/interfaces.go](cmd/practice/interfaces.go) | El **mismo** servicio funcionando con dos implementaciones distintas. |

```powershell
go test -v ./internal/practice/notify/
```

**Conceptos:** receptor valor vs puntero, composición (embedding), mutex.

---

## Paso 4 — Concurrencia
| Archivo | Qué mirar |
|---|---|
| [internal/practice/concurrency/concurrency.go](internal/practice/concurrency/concurrency.go) | `RunAll`: goroutines + channel + `context`. `SleepTask`: el `select`. `Counter`: mutex. |
| [internal/practice/concurrency/concurrency_test.go](internal/practice/concurrency/concurrency_test.go) | `TestRunAll_TimeoutCancelsTasks`: **el ejercicio que pide el PDF** (cancelar con `context.WithTimeout`). |
| [cmd/practice/concurrency.go](cmd/practice/concurrency.go) | Mira cuánto tarda: las tareas corren en paralelo. |

```powershell
go test -v ./internal/practice/concurrency/
```

**Conceptos:** goroutines, channels, `select`, `context`, goroutine leak, condición de carrera, channels vs memoria compartida.

---

## Paso 5 — Todo junto
| Archivo | Qué mirar |
|---|---|
| [cmd/practice/main.go](cmd/practice/main.go) | Un `main` delgado que solo llama a cada demo. |

```powershell
go run ./cmd/practice
go test ./...
```

---

## ✅ Autoevaluación (antes de pasar a la Jornada 2)

Debes poder explicar, sin mirar:

- [ ] Diferencia entre módulo y paquete, y para qué sirve `internal/`.
- [ ] Por qué en Go los errores son valores y no excepciones.
- [ ] Diferencia entre `errors.Is` y `errors.As`.
- [ ] Qué pasa si uso `%v` en lugar de `%w`.
- [ ] Cuándo usar `error` y cuándo `panic`.
- [ ] Dónde se define una interfaz y por qué debe ser pequeña.
- [ ] Cuándo usar receptor por puntero.
- [ ] Qué es la composición y por qué Go no usa herencia.
- [ ] Qué hace `context` y por qué va como primer parámetro.
- [ ] Qué es una condición de carrera y cómo se evita.
- [ ] Cuándo usar channels y cuándo mutex.
- [ ] Cuándo la concurrencia ayuda y cuándo solo complica.

Las respuestas están en [docs/conceptosJ1.md](docs/conceptosJ1.md).

---
---

# Orden de estudio — Jornada 2 (API de pedidos con Gin)

Antes de empezar, mira el mapa de la arquitectura en [docs/decisiones.md](docs/decisiones.md) (D1).
Lee **de adentro hacia afuera**: primero las reglas, al final HTTP.

```
cmd/api/main.go  →  arma todo
        │
adapters/httpapi  (HTTP, Gin)      adapters/repository/memory  (guardar)
        │                                   │
        └──────────►  application  ◄────────┘   (casos de uso)
                           │
                     domain/order               (reglas del negocio)
```

## Paso 1 — Dominio: las reglas
| Archivo | Qué mirar |
|---|---|
| [internal/domain/order/order.go](internal/domain/order/order.go) | `New` (crear válido), `Confirm` (la regla: no confirmar vacío ni dos veces), errores del dominio. **No importa Gin.** |
| [internal/domain/order/order_test.go](internal/domain/order/order_test.go) | Las reglas probadas sin HTTP ni base de datos. |

```powershell
go test -v ./internal/domain/...
```

## Paso 2 — Aplicación: los casos de uso
| Archivo | Qué mirar |
|---|---|
| [internal/application/order_service.go](internal/application/order_service.go) | `Confirm`: buscar → regla del dominio → guardar. La interfaz `OrderRepository` definida aquí. |

## Paso 3 — Repositorio en memoria
| Archivo | Qué mirar |
|---|---|
| [internal/adapters/repository/memory/order_repository.go](internal/adapters/repository/memory/order_repository.go) | Cumple `OrderRepository` sin mencionarlo. `RWMutex` y `clone`. |

## Paso 4 — Adaptador HTTP (Gin)
Lee en este orden:

| Archivo | Qué mirar |
|---|---|
| [internal/adapters/httpapi/dto.go](internal/adapters/httpapi/dto.go) | El JSON de entrada y salida. Etiquetas `json` y `binding`. |
| [internal/adapters/httpapi/handlers.go](internal/adapters/httpapi/handlers.go) | Los 3 endpoints. `ShouldBindJSON`. `c.Request.Context()`. |
| [internal/adapters/httpapi/errors.go](internal/adapters/httpapi/errors.go) | **Error del dominio → código HTTP.** El 500 no expone detalles. |
| [internal/adapters/httpapi/middleware.go](internal/adapters/httpapi/middleware.go) | El request ID / correlation ID. |
| [internal/adapters/httpapi/router.go](internal/adapters/httpapi/router.go) | Rutas y middlewares. |
| [internal/adapters/httpapi/handlers_test.go](internal/adapters/httpapi/handlers_test.go) | Pruebas con `httptest`: éxito, 400, 404, 409, 422, 500. |

```powershell
go test -v ./internal/adapters/httpapi/
```

## Paso 5 — Arrancar la API
| Archivo | Qué mirar |
|---|---|
| [cmd/api/main.go](cmd/api/main.go) | Arma las piezas (inyección de dependencias) y el apagado controlado. |

En una terminal:
```powershell
go run ./cmd/api
```
En **otra** terminal, pruébala:
```powershell
# Crear un pedido (copia el "id" de la respuesta)
Invoke-RestMethod -Method Post -Uri http://localhost:8080/orders -ContentType "application/json" -Body '{"customer_id":"c1","items":[{"sku":"SKU-1","quantity":2}]}'

# Consultarlo (cambia TU_ID)
Invoke-RestMethod http://localhost:8080/orders/TU_ID

# Confirmarlo (la segunda vez da 409)
Invoke-RestMethod -Method Post -Uri http://localhost:8080/orders/TU_ID/confirm

# Pedido vacío: créalo y al confirmarlo da 422
Invoke-RestMethod -Method Post -Uri http://localhost:8080/orders -ContentType "application/json" -Body '{"customer_id":"c1","items":[]}'
```
Para detener la API: `Ctrl + C` (verás "apagando la API...").

## Paso 6 — Decisiones
Lee [docs/decisiones.md](docs/decisiones.md): por qué 409 vs 422, por qué `POST .../confirm`, el límite del repositorio en memoria.

---

## ✅ Autoevaluación Jornada 2 (Revisión 1)

- [ ] ¿Por qué el dominio no importa Gin?
- [ ] Diferencia entre 400, 404, 409, 422 y 500.
- [ ] Diferencia entre entrada mal formada y violación de una regla del negocio. ¿Qué capa valida cada una?
- [ ] ¿Qué es un DTO y por qué no uso el struct del dominio directamente en el JSON?
- [ ] ¿Qué es la idempotencia y qué métodos HTTP deben serlo?
- [ ] ¿Qué es un middleware? Da un ejemplo del proyecto.
- [ ] ¿Para qué sirve el correlation ID?
- [ ] Diferencia entre `gin.Context` y `context.Context`. ¿Cuál se pasa a la aplicación?
- [ ] ¿Por qué un 500 no debe mostrar el error real?
- [ ] ¿Dónde pondrías la validación de un JWT y por qué?

Las respuestas están en [docs/conceptosJ2.md](docs/conceptosJ2.md).

---
---

# Orden de estudio — Jornada 3 (DDD)

**Qué cambió respecto a la J2:** el dominio pasó de "solo datos" a **proteger sus reglas**. Antes cualquiera podía escribir `o.Status = "CONFIRMED"`. Ahora los campos son privados y solo se cambian con métodos que validan.

## Paso 1 — Value Object: `Money`
| Archivo | Qué mirar |
|---|---|
| [internal/domain/order/money.go](internal/domain/order/money.go) | Campo privado, `NewMoney` rechaza negativos, `Add` devuelve uno nuevo (inmutable). El `panic` en `Times`. |
| [internal/domain/order/money_test.go](internal/domain/order/money_test.go) | Igualdad por valor e inmutabilidad probadas. |

## Paso 2 — `OrderItem`
| Archivo | Qué mirar |
|---|---|
| [internal/domain/order/item.go](internal/domain/order/item.go) | `NewOrderItem` valida. El doble `%w`. `isZero()` contra `OrderItem{}`. |

## Paso 3 — El agregado `Order`
| Archivo | Qué mirar |
|---|---|
| [internal/domain/order/order.go](internal/domain/order/order.go) | Comentario de `Order` (agregado, entidad, invariantes). Campos privados + getters. `Items()` devuelve copia. `New` vs `Restore`. `AddItem` y `Confirm`. |
| [internal/domain/order/order_test.go](internal/domain/order/order_test.go) | `TestConfirm`: el pedido vacío **sigue PENDING**. `TestItems_ReturnsCopy`: no se puede modificar desde afuera. |

```powershell
go test -v ./internal/domain/...
```

## Paso 4 — Caso de uso
| Archivo | Qué mirar |
|---|---|
| [internal/application/order_service.go](internal/application/order_service.go) | `Create` convierte datos simples en objetos del dominio. `Confirm`: si la regla falla, **no guarda**. |
| [internal/application/order_service_test.go](internal/application/order_service_test.go) | El error se propaga; un fallo de la BD se envuelve. |

## Paso 5 — Repositorio
| Archivo | Qué mirar |
|---|---|
| [internal/adapters/repository/memory/order_repository.go](internal/adapters/repository/memory/order_repository.go) | `orderRecord` (como una fila de BD). `toRecord` / `fromRecord`. El `%v` a propósito. |

## Paso 6 — HTTP (cambio pequeño)
| Archivo | Qué mirar |
|---|---|
| [internal/adapters/httpapi/dto.go](internal/adapters/httpapi/dto.go) | `toInput` y `toOrderResponse` ahora usan getters. |

## Paso 7 — Decisiones
[docs/decisiones.md](docs/decisiones.md): D5 (Money), D9 (agregado), D10 (repositorio).

---

## ✅ Autoevaluación Jornada 3

- [ ] ¿Por qué tener una carpeta `domain/` no significa que estás haciendo DDD?
- [ ] Diferencia entre entidad y value object. Da un ejemplo de cada uno del proyecto.
- [ ] ¿Qué es un agregado y cuál es la raíz en el proyecto?
- [ ] ¿Qué es una invariante? Nombra dos del proyecto.
- [ ] ¿Por qué los campos de `Order` son privados?
- [ ] ¿Qué es un repositorio y por qué su interfaz está en `application`?
- [ ] Diferencia entre errores de transporte y errores de dominio. ¿Quién detecta cada uno?
- [ ] ¿Por qué `fromRecord` usa `%v` y no `%w`?
- [ ] ¿Qué es un bounded context?
- [ ] ¿Cómo evitarías que dos `confirm` simultáneos se pisen?

Las respuestas están en [docs/conceptosJ3.md](docs/conceptosJ3.md).

---
---

# Orden de estudio — Jornada 4 (Integración con el catálogo)

**Qué cambió:** al crear un pedido, la API ya **no confía en el precio del cliente**: se lo pregunta a otro servicio, el **catálogo**. Y si el catálogo falla, la API no se cae.

```
Cliente → API (Create) → CatalogClient ──► catálogo (otro servicio, :8081)
                              │                 lento / caído / 404
                              └── timeout + reintentos + error controlado (503)
```

## Paso 1 — El puerto (la interfaz)
| Archivo | Qué mirar |
|---|---|
| [internal/application/catalog.go](internal/application/catalog.go) | `CatalogClient`, `Product` y los 2 errores: `ErrProductNotFound`, `ErrCatalogUnavailable`. |
| [internal/application/order_service.go](internal/application/order_service.go) | `Create` consulta el precio en el catálogo. `ItemInput` ya no tiene precio. |

## Paso 2 — El adaptador HTTP (lo más importante de la J4)
| Archivo | Qué mirar |
|---|---|
| [internal/adapters/catalog/http_client.go](internal/adapters/catalog/http_client.go) | `GetProduct`: el ciclo de reintentos. `getOnce`: timeout por intento y qué se reintenta según el código. `backoff`: exponencial + jitter. |
| [internal/adapters/catalog/http_client_test.go](internal/adapters/catalog/http_client_test.go) | Con `httptest.Server`: éxito, 404 sin reintento, se recupera tras 2 fallas, catálogo caído, timeout, cancelación. |

```powershell
go test -v ./internal/adapters/catalog/
```

## Paso 3 — Catálogo en memoria y catálogo ficticio
| Archivo | Qué mirar |
|---|---|
| [internal/adapters/catalog/static.go](internal/adapters/catalog/static.go) | Cumple el **mismo puerto** sin red. Lo usan las pruebas y la API cuando no hay `CATALOG_URL`. |
| [cmd/catalog/main.go](cmd/catalog/main.go) | Un segundo servicio real, solo con `net/http`. `CATALOG_DELAY` lo hace lento. |

## Paso 4 — HTTP de la API
| Archivo | Qué mirar |
|---|---|
| [internal/adapters/httpapi/errors.go](internal/adapters/httpapi/errors.go) | Nuevos: `PRODUCT_NOT_FOUND` (422) y `CATALOG_UNAVAILABLE` (503). |
| [internal/adapters/httpapi/handlers_test.go](internal/adapters/httpapi/handlers_test.go) | `TestCreateOrder_CatalogErrors`: catálogo caído → 503, y el cliente no puede poner el precio. |
| [cmd/api/main.go](cmd/api/main.go) | `newCatalog`: elige memoria o HTTP según `CATALOG_URL`. |

## Paso 5 — Pruébalo con dos servicios
```powershell
# Terminal 1: el catálogo
go run ./cmd/catalog

# Terminal 2: la API apuntando al catálogo
$env:CATALOG_URL="http://localhost:8081"; go run ./cmd/api

# Terminal 3: crear un pedido (el precio lo pone el catálogo)
Invoke-RestMethod -Method Post -Uri http://localhost:8080/orders -ContentType "application/json" -Body '{"customer_id":"c1","items":[{"sku":"SKU-2","quantity":3}]}'
```
Experimentos:
- **Apaga el catálogo** (Ctrl+C en la terminal 1) y vuelve a crear → **503**.
- **Catálogo lento:** `$env:CATALOG_DELAY="5s"; go run ./cmd/catalog` → **503** en ~3 s, sin colgarse.
- **SKU inexistente** (`"sku":"NOPE"`) → **422**.

## Paso 6 — Decisiones
[docs/decisiones.md](docs/decisiones.md): D11 (precio del catálogo), **D12 (HTTP vs gRPC vs SOAP vs colas)**, D13 (timeouts y reintentos). Resumen en [README.md](README.md#decisiones-de-diseño).

---

## ✅ Autoevaluación Jornada 4 (Revisión 2)

- [ ] Diferencia entre comunicación síncrona y asíncrona. ¿Cuál usamos con el catálogo y por qué?
- [ ] ¿Para qué sirve un timeout? ¿Qué pasaría sin él?
- [ ] ¿Qué errores se reintentan y cuáles no? ¿Por qué un 404 no?
- [ ] ¿Por qué se puede reintentar un GET y no un POST que cobra?
- [ ] ¿Qué es backoff exponencial y para qué sirve el jitter?
- [ ] ¿Qué es un circuit breaker?
- [ ] ¿Por qué el catálogo caído da 503 y no 500?
- [ ] ¿Por qué el precio no lo envía el cliente?
- [ ] ¿Cuándo usarías gRPC, SOAP o una cola en vez de HTTP?
- [ ] Diferencia entre cola y pub/sub.
- [ ] ¿Qué es la consistencia eventual?

Las respuestas están en [docs/conceptosJ4.md](docs/conceptosJ4.md).

---
---

# Orden de estudio — Jornada 5 (Docker y Kubernetes)

**Qué se agregó:** la API ahora se puede **empaquetar en un contenedor** y **desplegar en Kubernetes**, con health checks para que Kubernetes sepa cuándo reiniciarla y cuándo enviarle tráfico.

```
Dockerfile ──build──► imagen orders-api:dev ──► Kubernetes
                                                  ├─ Deployment (pods) ─ /healthz /readyz
                                                  ├─ Service (dirección fija)
                                                  └─ ConfigMap (configuración)
```

## Paso 1 — Health checks (código Go)
| Archivo | Qué mirar |
|---|---|
| [internal/adapters/httpapi/health.go](internal/adapters/httpapi/health.go) | `/healthz` vs `/readyz`. El comentario de **por qué no revisan el catálogo**. |
| [internal/adapters/httpapi/health_test.go](internal/adapters/httpapi/health_test.go) | Arrancando → vivo pero no listo; en marcha → listo; apagándose → ya no listo. |
| [cmd/api/main.go](cmd/api/main.go) | `ready.SetReady(true)` al arrancar; al apagarse `SetReady(false)` + `SHUTDOWN_DELAY` (drenado). |

```powershell
go test -v -run Health ./internal/adapters/httpapi/
```

## Paso 2 — Docker
| Archivo | Qué mirar |
|---|---|
| [Dockerfile](Dockerfile) | Las 2 etapas. Por qué `go.mod` se copia primero (caché). `CGO_ENABLED=0`. Distroless y `USER 65532`. |
| [.dockerignore](.dockerignore) | Qué NO entra a la imagen (el PDF, `.git`...). |

## Paso 3 — Kubernetes (lee en este orden)
| Archivo | Qué mirar |
|---|---|
| [deploy/k8s/namespace.yaml](deploy/k8s/namespace.yaml) | El "espacio" del proyecto. |
| [deploy/k8s/configmap.yaml](deploy/k8s/configmap.yaml) | Configuración fuera de la imagen. `CATALOG_URL: http://catalog`. |
| [deploy/k8s/api.yaml](deploy/k8s/api.yaml) | **El más importante:** Deployment (réplicas, rolling update, requests/limits, probes, seguridad) + Service. |
| [deploy/k8s/catalog.yaml](deploy/k8s/catalog.yaml) | El catálogo con **2 réplicas** (no guarda estado). |
| [deploy/k8s/kustomization.yaml](deploy/k8s/kustomization.yaml) | Aplica todo junto. |

## Paso 4 — Ejecutarlo (cuando instales las herramientas)

**Instalar** (una sola vez):
```powershell
winget install --id Docker.DockerDesktop -e   # incluye docker y kubectl
winget install --id Kubernetes.kind -e
```
Reinicia el PC, abre Docker Desktop y espera a que diga "Engine running".

**A. Correr la API en un contenedor:**
```powershell
docker build -t orders-api:dev .
docker run --rm -p 8080:8080 orders-api:dev
# en otra terminal:
Invoke-RestMethod http://localhost:8080/readyz
```

**B. Desplegar en Kubernetes local (kind):**
```powershell
kind create cluster --name orders
docker build -t orders-api:dev .
docker build -t catalog:dev --build-arg TARGET=catalog .
kind load docker-image orders-api:dev catalog:dev --name orders   # pasa las imágenes al clúster

kubectl apply --dry-run=client -k deploy/k8s   # validar los manifiestos (client: el namespace aún no existe)
kubectl apply -k deploy/k8s                    # desplegar
kubectl get pods -n orders                     # ver el estado de los pods
kubectl logs -n orders deployment/orders-api   # ver los logs

kubectl port-forward -n orders service/orders-api 8080:80   # abrir la API en localhost:8080
```

**C. Experimentos para la entrevista:**
- **Catálogo caído:** `kubectl scale -n orders deployment/catalog --replicas=0` → crear un pedido da **503**, pero la API **no se reinicia** (liveness no revisa el catálogo).
- **Rolling update:** `kubectl rollout restart -n orders deployment/orders-api` y luego `kubectl get pods -n orders -w`: el pod nuevo arranca antes de que se apague el viejo.
- **Rollback:** `kubectl rollout undo -n orders deployment/orders-api`.

**Borrar todo:** `kind delete cluster --name orders`

## Paso 5 — Decisiones
[docs/decisiones.md](docs/decisiones.md): D14 (por qué 1 réplica), D15 (qué revisa cada probe), D16 (distroless sin root).

---

## ✅ Autoevaluación Jornada 5

- [ ] Diferencia entre imagen y contenedor.
- [ ] ¿Qué es un Dockerfile multistage y qué ventaja tiene?
- [ ] ¿Qué es un Pod, un Deployment y un Service?
- [ ] ¿Para qué sirve un ConfigMap? ¿Y un Secret? ¿Por qué un Secret de K8s no es suficiente?
- [ ] Diferencia entre liveness y readiness. ¿Por qué liveness no revisa el catálogo?
- [ ] ¿Qué son requests y limits? ¿Qué pasa si se pasa el límite de memoria?
- [ ] ¿Cómo funciona un rolling update sin caída? ¿Cómo haces rollback?
- [ ] ¿Qué es HPA y qué necesita?
- [ ] ¿Por qué la API tiene 1 réplica y el catálogo 2?
- [ ] ¿Qué problemas **no** resuelve Docker?

Las respuestas están en [docs/conceptosJ5.md](docs/conceptosJ5.md).

---
---

# Orden de estudio — Jornada 6 (DevSecOps, IaC y observabilidad)

**Qué se agregó:** verificación automática del código (CI), seguridad en el pipeline, logs JSON que se pueden seguir entre servicios, y un ejemplo de Terraform.

## Paso 1 — Logs estructurados y correlation ID
| Archivo | Qué mirar |
|---|---|
| [internal/platform/requestid/requestid.go](internal/platform/requestid/requestid.go) | Guardar el ID en el `context.Context`. Por qué la clave es un tipo privado. |
| [internal/platform/logging/logging.go](internal/platform/logging/logging.go) | JSON vs texto. `contextHandler`: agrega el `correlation_id` a cada log solo. |
| [internal/adapters/httpapi/middleware.go](internal/adapters/httpapi/middleware.go) | `RequestID` ahora guarda el ID en el context. `AccessLog`: un log por solicitud. |
| [internal/adapters/httpapi/router.go](internal/adapters/httpapi/router.go) | Por qué el **orden** de los middlewares importa. |
| [internal/adapters/catalog/http_client.go](internal/adapters/catalog/http_client.go) | Envía `X-Request-ID` al catálogo. Log de cada reintento. |
| [internal/adapters/httpapi/middleware_test.go](internal/adapters/httpapi/middleware_test.go) | La evidencia: una solicitud = una línea JSON interpretable. |

Pruébalo:
```powershell
go run ./cmd/catalog                                            # terminal 1
$env:CATALOG_URL="http://localhost:8081"; go run ./cmd/api       # terminal 2
# terminal 3:
Invoke-RestMethod -Method Post -Uri http://localhost:8080/orders -ContentType "application/json" -Headers @{"X-Request-ID"="demo-1"} -Body '{"customer_id":"c1","items":[{"sku":"SKU-1","quantity":2}]}'
```
Busca `demo-1` en los logs de **las dos** terminales. Luego apaga el catálogo y repite: verás los `reintentando catálogo`.
Para logs legibles en local: `$env:LOG_FORMAT="text"`.

## Paso 2 — CI y seguridad
| Archivo | Qué mirar |
|---|---|
| [.github/workflows/ci.yml](.github/workflows/ci.yml) | Los 3 jobs: calidad, dependencias (govulncheck), imagen (Trivy). `permissions: read`. |
| [.golangci.yml](.golangci.yml) | Qué linters se usan; `gosec` es el SAST. |
| [scripts/verify.ps1](scripts/verify.ps1) | La versión local del CI. |

```powershell
.\scripts\verify.ps1 -Race
```
> El CI de GitHub correrá cuando el repositorio esté en GitHub. Localmente ya pasan formato, vet, pruebas y `-race`.

## Paso 3 — Terraform (IaC)
| Archivo | Qué mirar |
|---|---|
| [deploy/terraform/versions.tf](deploy/terraform/versions.tf) | **Provider** y comentario sobre el **state** remoto. |
| [deploy/terraform/variables.tf](deploy/terraform/variables.tf) | **Variables** con validación. |
| [deploy/terraform/main.tf](deploy/terraform/main.tf) | **Resources** declarativos y dependencias implícitas. |
| [deploy/terraform/outputs.tf](deploy/terraform/outputs.tf) | Lo que muestra al terminar. |

Pruébalo (Terraform corre dentro de Docker, no hay que instalarlo):
```powershell
docker build -t orders-api:dev . ; docker build -t catalog:dev --build-arg TARGET=catalog .
$tf = "docker run --rm -v ${PWD}/deploy/terraform:/tf -v /var/run/docker.sock:/var/run/docker.sock -w /tf hashicorp/terraform:1.9"
Invoke-Expression "$tf init"
Invoke-Expression "$tf plan"                  # qué haría
Invoke-Expression "$tf apply -auto-approve"   # lo hace → API en http://localhost:8097
Invoke-Expression "$tf state list"            # lo que recuerda
Invoke-Expression "$tf destroy -auto-approve" # lo borra
```

## Paso 4 — La nota de observabilidad
[docs/observabilidad.md](docs/observabilidad.md): conceptos de IaC, **qué mediría la API** (4 señales doradas), **dónde instrumentar con OpenTelemetry**, instrumentar vs recolectar vs almacenar vs visualizar, y **cómo diagnosticar un aumento de latencia**.

## Paso 5 — Decisiones
[docs/decisiones.md](docs/decisiones.md): D17 (logs), D18 (pipeline), D19 (Terraform con Docker).

---

## ✅ Autoevaluación Jornada 6

- [ ] ¿Qué es CI y qué pasos tiene nuestro pipeline?
- [ ] Diferencia entre SAST, análisis de dependencias y escaneo de imágenes. Una herramienta de cada uno.
- [ ] ¿Por qué los secretos no van en Git? ¿Dónde van?
- [ ] ¿Qué es IaC y qué significa "declarativo"?
- [ ] ¿Qué es el state de Terraform y por qué va en un backend remoto con bloqueo?
- [ ] Diferencia entre `plan` y `apply`.
- [ ] ¿Qué es un log estructurado y por qué es mejor que texto libre?
- [ ] ¿Cómo se sigue una solicitud entre la API y el catálogo?
- [ ] Diferencia entre logs, métricas y trazas.
- [ ] ¿Cuáles son las 4 señales doradas?
- [ ] Instrumentar vs recolectar vs almacenar vs visualizar.
- [ ] La latencia subió en producción: ¿qué miras primero y en qué orden?

Las respuestas están en [docs/conceptosJ6.md](docs/conceptosJ6.md) y [docs/observabilidad.md](docs/observabilidad.md).

---
---

# Orden de estudio — Jornada 7 (IA y demo final)

**Qué se agregó:** un clasificador de pedidos con IA **simulada**, diseñado para que la IA **nunca rompa la API**; el README profesional; y el guion de la demo.

```
GET /orders/:id/classification
        │
ClassificationService ──► modelo IA (Simulated) ── timeout 1s ── ¿respuesta válida?
        │                                                           sí → source: model
        └──────────────────► reglas fijas (Rules) ◄── no / falló / tardó → source: rules
```

## Paso 1 — El puerto y el caso de uso
| Archivo | Qué mirar |
|---|---|
| [internal/application/classification.go](internal/application/classification.go) | `OrderClassifier`, etiquetas permitidas, `ClassificationInput` **sin el cliente** (privacidad). `askModel`: timeout + log + validación. `Classify`: el fallback. |
| [internal/application/classification_test.go](internal/application/classification_test.go) | Los 4 casos de falla caen a reglas. `spyClassifier`: qué datos llegan al modelo. |

## Paso 2 — Los adaptadores
| Archivo | Qué mirar |
|---|---|
| [internal/adapters/classifier/rules.go](internal/adapters/classifier/rules.go) | Reglas deterministas: siempre la misma respuesta, confianza 1. |
| [internal/adapters/classifier/simulated.go](internal/adapters/classifier/simulated.go) | El "modelo": latencia, confianza que baja en casos dudosos y los modos `fail`, `invalid`, `slow`. |

## Paso 3 — HTTP
| Archivo | Qué mirar |
|---|---|
| [internal/adapters/httpapi/classification.go](internal/adapters/httpapi/classification.go) | Por qué es un endpoint **aparte**. El campo `source`. |
| [internal/adapters/httpapi/router.go](internal/adapters/httpapi/router.go) | `Deps`: el router recibe un struct en vez de muchos parámetros. |
| [cmd/api/main.go](cmd/api/main.go) | `CLASSIFIER_MODE` para simular cada riesgo. |

Pruébalo:
```powershell
go run ./cmd/api
# en otra terminal:
$p = Invoke-RestMethod -Method Post -Uri http://localhost:8080/orders -ContentType "application/json" -Body '{"customer_id":"c1","items":[{"sku":"SKU-2","quantity":40}]}'
Invoke-RestMethod http://localhost:8080/orders/$($p.id)/classification     # BULK, source: model
```
Luego reinicia la API con `$env:CLASSIFIER_MODE="fail"` (o `invalid`, `slow`) y repite: `source: rules`. Prueba también `"quantity":16`: el modelo duda y ganan las reglas.

## Paso 4 — README y demo
| Archivo | Qué mirar |
|---|---|
| [README.md](README.md) | Ahora es el README **del proyecto**: qué hace, arquitectura, endpoints, errores, configuración, limitaciones. El plan del curso quedó en `docs/plan.md`, que es local (no se sube a GitHub porque copia un documento confidencial). |
| [docs/demo.md](docs/demo.md) | **El guion de los 10 minutos**, con comandos, qué decir y preguntas probables. **Ensáyalo al menos una vez.** |

## Paso 5 — Decisiones
[docs/decisiones.md](docs/decisiones.md): D20 (IA aislada y con respaldo), D21 (`Deps`).

---

## ✅ Autoevaluación Jornada 7

- [ ] ¿Cuándo usarías IA y cuándo una regla fija? Un ejemplo de cada una.
- [ ] ¿Cómo está aislada la IA del dominio en el proyecto?
- [ ] ¿Qué pasa si el modelo falla, tarda o alucina? ¿Qué responde la API?
- [ ] ¿Por qué la clasificación está en un endpoint aparte?
- [ ] ¿Qué validaciones se le hacen a la respuesta del modelo?
- [ ] ¿Qué datos se le envían al modelo y por qué no el cliente?
- [ ] Diferencia entre llamar a un LLM, RAG y un agente.
- [ ] ¿Qué son los embeddings?
- [ ] ¿Dónde van los controles de seguridad, la evaluación de respuestas y la observabilidad de la IA?
- [ ] En la demo: ¿qué limitaciones tiene el proyecto y cómo evolucionaría?

Las respuestas están en [docs/conceptosJ7.md](docs/conceptosJ7.md) y [docs/demo.md](docs/demo.md).
