# Jornada 6 — DevSecOps, IaC y observabilidad (resumen rápido)

---

## **CI (Integración Continua)**
Cada vez que subes código, un servidor **compila, prueba y revisa** todo automáticamente. Si algo falla, se marca en rojo antes de llegar a producción.
📌 `.github/workflows/ci.yml`

## **CD (Entrega/Despliegue Continuo)**
El paso siguiente: si el CI pasa, el código se **despliega solo** (a QA o a producción).

## **Pipeline**
La secuencia de pasos automáticos: formato → vet → linter → pruebas → seguridad → imagen.
📌 Local: `scripts/verify.ps1` hace lo mismo que el CI.

## **Linter**
Programa que revisa el código **sin ejecutarlo** buscando errores comunes, código muerto o malas prácticas.
📌 `golangci-lint` con la configuración `.golangci.yml`.

## **Cobertura de pruebas**
Qué porcentaje del código ejecutan las pruebas. No mide calidad, pero muestra **qué no se prueba**.
📌 Dominio 84%, HTTP 98.5%, repositorio 97.5%.

## **Detector de carreras (`-race`)**
Ejecuta las pruebas vigilando si dos goroutines tocan la misma variable sin protección.
📌 Pasa en todo el proyecto (corre en Docker con `verify.ps1 -Race`).

## **DevSecOps**
Meter la **seguridad dentro del pipeline**, desde el inicio, en vez de revisarla al final.

## **SAST (análisis estático de seguridad)**
Revisa **tu código** buscando fallas de seguridad (SQL injection, aleatoriedad débil, timeouts faltantes).
📌 `gosec` dentro de golangci-lint.

## **Análisis de dependencias (SCA)**
Revisa **las librerías que usas** contra bases de vulnerabilidades conocidas (CVE).
📌 `govulncheck`: solo avisa si tu código **realmente llama** a la función vulnerable.

## **Escaneo de imágenes**
Revisa **la imagen Docker completa**: el sistema base y los binarios.
📌 `Trivy` en el job `image` del CI.

## **Caso real del proyecto**
govulncheck encontró **5 vulnerabilidades** en librerías que trae Gin (no en nuestro código). Se actualizaron `x/net` y `quic-go`, las pruebas siguieron pasando y el nuevo escaneo dio **0**. Trivy: **0** en la imagen. golangci-lint: **0 issues**.

## **SAST vs dependencias vs imágenes (pregunta clave)**
SAST = **mi código**. Dependencias = **código de otros que importo**. Imagen = **todo lo que viaja en el contenedor**.

## **Gestión de secretos**
Las contraseñas y tokens **nunca** van en el código ni en Git. Se guardan en un gestor (Vault, AWS Secrets Manager) y se inyectan al ejecutar.
📌 `.env`, `*.tfstate` y `*.tfvars` están en `.gitignore`.

## **Principio de menor privilegio**
Dar **solo los permisos necesarios**.
📌 El CI solo puede leer (`permissions: contents: read`); el contenedor corre sin root.

## **IaC (Infraestructura como código)**
La infraestructura (servidores, redes, bases de datos) se define en **archivos versionados**, no a mano en una consola. Reproducible y revisable.
📌 `deploy/terraform/`

## **Declarativo**
Describes **el resultado** ("quiero 2 contenedores"), no los pasos. La herramienta calcula cómo llegar.

## **Provider / Resource / Variables**
**Provider** = plugin de la plataforma (Docker, AWS). **Resource** = cada cosa que debe existir. **Variables** = lo que cambia por entorno.

## **State**
El archivo donde Terraform recuerda **qué creó**. En equipo va en un backend **remoto con bloqueo**. Puede tener secretos → nunca en Git.

## **Plan / Apply**
**Plan** = vista previa, no cambia nada. **Apply** = ejecuta y actualiza el state.
📌 Verificado: plan (5 por crear) → apply → plan otra vez = "No changes" → destroy.

## **Observabilidad**
Poder entender **qué pasa adentro** del sistema mirando lo que produce: **logs, métricas y trazas**.

## **Log estructurado**
Cada log es un **JSON con campos**, no texto libre. Se puede filtrar y agrupar.
📌 `{"msg":"http request","route":"/orders","status":503,"duration_ms":373,"correlation_id":"demo-456"}`

## **Correlation ID entre servicios**
El mismo ID viaja de la API al catálogo (header `X-Request-ID`): una solicitud se sigue **en todos los servicios**.
📌 `demo-123` aparece en los logs de la API **y** del catálogo.

## **Métricas**
**Números en el tiempo**: solicitudes por segundo, % de errores, latencia p95. Baratas de guardar; sirven para **alertas**.

## **Trazas (tracing)**
El **recorrido completo** de una solicitud, dividido en tramos (*spans*): handler → caso de uso → llamada al catálogo. Muestra **dónde** se va el tiempo.

## **Las 4 señales doradas**
**Latencia** (cuánto tarda), **tráfico** (cuánta carga), **errores** (cuánto falla), **saturación** (qué tan lleno está).

## **p95 / p99**
"El 95% de las solicitudes tarda menos que X". El **promedio esconde** a los usuarios lentos; el p95 no.

## **OpenTelemetry (OTel)**
Estándar abierto para **producir y enviar** logs, métricas y trazas, sin casarte con un proveedor.

## **Instrumentar → recolectar → almacenar → visualizar**
**Instrumentar** = el código genera datos (SDK OTel). **Recolectar** = el Collector los recibe y reenvía. **Almacenar** = Prometheus, Tempo, Loki. **Visualizar** = Grafana (dashboards y alertas).

## **Diagnosticar un aumento de latencia**
**Métricas** dicen *cuándo y dónde* (qué ruta subió el p95). **Trazas** dicen *qué parte* (¿el catálogo?). **Logs** dicen *por qué* (reintentos, timeouts).

---

## 🎯 **Frase para la entrevista**
> "Tengo un pipeline de CI con formato, vet, golangci-lint con gosec, pruebas con `-race` y cobertura, govulncheck para dependencias y Trivy para la imagen. Los logs son JSON con un correlation ID que viaja entre servicios. Definí las métricas según las 4 señales doradas y dónde instrumentar con OpenTelemetry, enviando a un Collector que reparte a Prometheus, Tempo y Loki. Y tengo un ejemplo de Terraform donde apliqué el ciclo plan → apply → destroy con su state."
