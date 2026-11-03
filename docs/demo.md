# Guion de la demo final (10 minutos)

Lo que pide el PDF: ejecutar el caso principal, mostrar pruebas, reglas de negocio, un fallo de integración, el contenedor y evidencias de operación; y responder **por qué**, **qué limitaciones** y **cómo evolucionaría**.

---

## Antes de empezar (5 min antes, no cuenta en la demo)

```powershell
cd C:\Users\RIWI\Desktop\BackendGo
docker build -t orders-api:dev . ; docker build -t catalog:dev --build-arg TARGET=catalog .
```
Abre **3 terminales** y deja Docker Desktop corriendo. En VS Code ten abiertos: `internal/domain/order/order.go`, `internal/adapters/httpapi/errors.go`, `internal/adapters/catalog/http_client.go`, `internal/application/classification.go`.

---

## 1. Qué es y cómo está hecho — 1 min

Muestra el diagrama del [README.md](../README.md#arquitectura) y di:

> "Es una API de pedidos en Go con Gin. Usé arquitectura hexagonal con DDD: el dominio tiene las reglas y no depende de nada externo; la aplicación tiene los casos de uso y define interfaces; y los adaptadores implementan HTTP, la persistencia, el catálogo y la IA. Así puedo probar las reglas solas y cambiar cualquier tecnología sin tocarlas."

## 2. Caso principal + reglas de negocio — 2 min

**Terminal 1:**
```powershell
go run ./cmd/catalog
```
**Terminal 2:**
```powershell
$env:CATALOG_URL="http://localhost:8081"; $env:LOG_FORMAT="text"; go run ./cmd/api
```
**Terminal 3:**
```powershell
$p = Invoke-RestMethod -Method Post -Uri http://localhost:8080/orders -ContentType "application/json" -Body '{"customer_id":"c1","items":[{"sku":"SKU-1","quantity":2}]}'
$p                                                                       # 201: total 3000, el precio lo puso el catálogo
Invoke-RestMethod -Method Post -Uri http://localhost:8080/orders/$($p.id)/confirm   # 200: CONFIRMED
Invoke-RestMethod -Method Post -Uri http://localhost:8080/orders/$($p.id)/confirm   # 409: ya confirmado

$v = Invoke-RestMethod -Method Post -Uri http://localhost:8080/orders -ContentType "application/json" -Body '{"customer_id":"c1","items":[]}'
Invoke-RestMethod -Method Post -Uri http://localhost:8080/orders/$($v.id)/confirm   # 422: ORDER_EMPTY
```
Abre `order.go` → método `Confirm`:
> "La regla vive en el dominio. Los campos son privados, así que nadie puede poner el pedido en CONFIRMED saltándosela. El handler solo traduce el error a HTTP en un único lugar, `errors.go`: 409 porque el estado no lo permite, 422 porque el contenido viola la regla."

## 3. Fallo de integración — 2 min

En la **terminal 1**, `Ctrl + C` (apagas el catálogo). En la **terminal 3**:
```powershell
Invoke-RestMethod -Method Post -Uri http://localhost:8080/orders -ContentType "application/json" -Headers @{"X-Request-ID"="demo-falla"} -Body '{"customer_id":"c1","items":[{"sku":"SKU-1","quantity":1}]}'
# 503 CATALOG_UNAVAILABLE en menos de 1 segundo
```
Muestra en la **terminal 2** los logs con `demo-falla`: dos `reintentando catálogo` y el 503.
> "El catálogo es una dependencia síncrona. Cada intento tiene timeout, reintento solo fallas temporales con backoff exponencial porque el GET es idempotente, y si se agota respondo 503 controlado. El resto de la API sigue funcionando: es una falla parcial."

Vuelve a encender el catálogo (`go run ./cmd/catalog`).

## 4. IA aislada — 1.5 min

```powershell
Invoke-RestMethod http://localhost:8080/orders/$($p.id)/classification   # source: model
```
Detén la API (terminal 2) y arráncala con la IA "alucinando":
```powershell
$env:CLASSIFIER_MODE="invalid"; go run ./cmd/api
```
Crea un pedido y clasifícalo otra vez → `source: rules`. Muestra el log `etiqueta desconocida "URGENTE_VIP"`.
> "La IA es un complemento, no una regla del negocio: va en un endpoint aparte y confirmar nunca depende de ella. Valido su respuesta (etiqueta permitida, confianza mínima) y, si falla, tarda o alucina, uso reglas fijas. Al modelo solo le envío lo necesario, sin datos del cliente."

Quita el modo: `Remove-Item Env:CLASSIFIER_MODE`.

## 5. Pruebas y seguridad — 1.5 min

```powershell
.\scripts\verify.ps1 -Race
```
> "Formato, vet, pruebas en todas las capas con cobertura y detector de carreras. El dominio se prueba sin Gin ni base de datos. En CI también corre golangci-lint con gosec, govulncheck y Trivy. govulncheck encontró 5 vulnerabilidades en dependencias indirectas de Gin; las actualicé y quedó en cero."

*(Si el tiempo es justo, muestra solo `go test ./...`.)*

## 6. Contenedor y operación — 1.5 min

```powershell
kind create cluster --name orders      # si no existe (tarda ~1 min: hazlo antes de la demo)
kind load docker-image orders-api:dev catalog:dev --name orders
kubectl apply -k deploy/k8s
kubectl get pods -n orders
kubectl scale -n orders deployment/catalog --replicas=0   # catálogo caído
kubectl get pods -n orders                                # la API sigue Running, 0 reinicios
```
> "Imagen distroless sin root de 36 MB. Liveness no revisa el catálogo para no provocar reinicios en cascada; readiness saca el pod del tráfico mientras arranca o se apaga. Rolling update sin caída. La API tiene una réplica porque guarda en memoria: para escalar, primero la base de datos."

## 7. Limitaciones y evolución — 30 s

Muestra la tabla **Limitaciones y cómo evolucionaría** del [README.md](../README.md#limitaciones-y-cómo-evolucionaría).

---

## Preguntas que probablemente te harán

**¿Por qué Go y Gin?** Go: binario estático pequeño, concurrencia simple, ideal para servicios. Gin: router rápido y middlewares; lo uso solo como adaptador, podría cambiarlo.

**¿Por qué no microservicios de verdad?** El PDF pide un flujo verificable. El catálogo como segundo servicio ya muestra integración, resiliencia y propagación del correlation ID sin la complejidad de 3 servicios y un broker.

**¿Qué pasa si dos personas confirman el mismo pedido a la vez?** Hoy podrían leer los dos PENDING (*lost update*). Con PostgreSQL usaría bloqueo optimista: columna `version` y `UPDATE ... WHERE version = ?`; si no actualiza, 409.

**¿Cuándo usarías IA y cuándo una regla fija?** Regla fija cuando la lógica se puede escribir, explicar y auditar (umbrales, montos). IA cuando hay ambigüedad o lenguaje natural (descripciones, reclamos) y un error ocasional es aceptable y reversible.

**¿Diferencia entre llamar a un LLM, RAG y un agente?** LLM = una pregunta, una respuesta. RAG = antes de preguntar busco documentos relevantes y se los paso como contexto. Agente = el modelo decide qué herramientas llamar en varios pasos; necesita más controles.

**¿Dónde van los controles de seguridad de la IA?** En el adaptador y en el caso de uso: minimizar datos, API key en un gestor de secretos, timeout, validar la salida, no ejecutar acciones sin aprobación y registrar cada llamada (latencia, confianza, fuente).

**Si mañana el negocio pide notificar al cliente al confirmar?** Publicaría un evento `OrderConfirmed` en una cola y un consumidor idempotente enviaría el correo: confirmar no debería esperar al correo (asíncrono).

**¿Qué mirarías si sube la latencia en producción?** Métricas (qué ruta subió el p95) → trazas (qué tramo tarda) → logs con el correlation ID (reintentos, timeouts). Ver [observabilidad.md](observabilidad.md).
