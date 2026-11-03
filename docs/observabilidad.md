# Nota: IaC y observabilidad

Nota corta que pide el PDF (J6): conceptos de IaC, qué mediría la API, dónde instrumentar con OpenTelemetry y cómo llevar la telemetría a un backend de monitoreo.

---

## 1. IaC (Infraestructura como código) — ejemplo en [deploy/terraform](../deploy/terraform/)

| Concepto | Qué es | En el ejemplo |
|---|---|---|
| **Declaración** | Describes el *resultado* que quieres, no los pasos | `main.tf`: "una red y dos contenedores" |
| **Provider** | Plugin que habla con la plataforma | `kreuzwerker/docker` en `versions.tf` |
| **Resource** | Cada cosa que debe existir | `docker_network`, `docker_container`... |
| **Variables** | Valores que cambian por entorno | `variables.tf`, `terraform.tfvars.example` |
| **State** | Registro de lo que Terraform creó y sus IDs reales | `terraform.tfstate` (local, fuera de Git) |
| **Plan** | Vista previa de los cambios, sin tocar nada | `Plan: 5 to add` |
| **Apply** | Ejecuta el plan y actualiza el state | `Apply complete! Resources: 5 added` |

**Verificado (2026-11-03):** `init` → `validate` → `plan` (5 por crear) → `apply` → la API respondió → segundo `plan` = *No changes* → `destroy`.

**Cuidados con el state:** puede contener datos sensibles y es la "verdad" de Terraform. En equipo va en un backend remoto (S3, GCS, Azure Blob, Terraform Cloud) **con bloqueo**, para que dos personas no apliquen a la vez.

---

## 2. Qué mediría la API

**Las 4 señales doradas**, aplicadas a este proyecto:

| Señal | Métrica | Por qué |
|---|---|---|
| **Tráfico** | solicitudes por segundo, por `route` y `method` | cuánta carga hay |
| **Errores** | % de respuestas 5xx (y 503 aparte) por `route` | si la API está fallando |
| **Latencia** | duración p50 / p95 / p99 por `route` | el promedio esconde a los usuarios lentos |
| **Saturación** | CPU y memoria del pod vs su *limit*; número de goroutines | qué tan cerca está del límite |

**De la dependencia (catálogo):** latencia de `GetProduct`, cantidad de **reintentos**, % de `ErrCatalogUnavailable`.

**Del negocio:** pedidos creados y confirmados por minuto, intentos de confirmar pedidos vacíos (`ORDER_EMPTY`).

> Hoy ya existe la base: cada solicitud deja un log JSON con `route`, `status`, `duration_ms` y `correlation_id` ([middleware.go](../internal/adapters/httpapi/middleware.go)), y cada reintento al catálogo deja un `WARN`.

---

## 3. Dónde instrumentar con OpenTelemetry (siguiente paso; no implementado)

| Lugar | Qué agregar | Da |
|---|---|---|
| [router.go](../internal/adapters/httpapi/router.go) | middleware `otelgin` | un *span* por solicitud + métricas HTTP automáticas |
| [http_client.go](../internal/adapters/catalog/http_client.go) | `otelhttp.NewTransport` en el `http.Client` | un *span hijo* por llamada al catálogo y propagación del contexto de traza (header `traceparent`) |
| [order_service.go](../internal/application/order_service.go) | `tracer.Start(ctx, "Confirm")` | ver cuánto tarda cada caso de uso |
| [logging.go](../internal/platform/logging/logging.go) | agregar `trace_id` al log | saltar de un log a su traza |
| [cmd/api/main.go](../cmd/api/main.go) | inicializar el SDK y el exportador OTLP | enviar todo al Collector |

**Instrumentar ≠ recolectar ≠ almacenar ≠ visualizar:**

```
 INSTRUMENTAR          RECOLECTAR            ALMACENAR               VISUALIZAR
 (en el código)        (procesa y envía)     (guarda y consulta)     (dashboards, alertas)

 SDK OpenTelemetry ──► OTel Collector ──┬──► Prometheus (métricas) ─┐
 en la API y el        (agrega, filtra, ├──► Tempo/Jaeger (trazas) ─┼──► Grafana
 catálogo              muestrea)        └──► Loki (logs) ───────────┘
```

- **Instrumentar:** el código produce la telemetría (spans, métricas, logs).
- **Recolectar:** un agente (OTel Collector) la recibe, la procesa y la reenvía. La app no sabe a qué backend va.
- **Almacenar:** bases especializadas para cada tipo de dato.
- **Visualizar:** dashboards y **alertas** (ej. "p95 > 500 ms durante 5 min").

En Kubernetes: el Collector corre como otro Deployment (o DaemonSet) y la API le envía por OTLP a `http://otel-collector:4317`.

---

## 4. ¿Cómo diagnosticaría un aumento de latencia en producción?

1. **Métricas** → ¿qué `route` subió su p95? ¿Desde cuándo? ¿Coincide con un despliegue? ¿Subió la CPU (saturación)?
2. **Trazas** → en una solicitud lenta, ¿qué *span* se lleva el tiempo: el handler o la llamada al catálogo?
3. **Logs** → con el `correlation_id` (o `trace_id`) de esa solicitud: ¿hubo **reintentos** al catálogo? ¿timeouts?

Ejemplo real de este proyecto: si el catálogo se pone lento, los logs muestran `reintentando catálogo` y la latencia de `POST /orders` sube hasta ~3 s (3 intentos × 1 s). Las métricas dicen **cuándo y dónde**, las trazas **qué parte** y los logs **por qué**.
