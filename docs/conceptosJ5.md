# Jornada 5 — Contenedores y Kubernetes (resumen rápido)

---

## **Docker**
Empaqueta tu aplicación **con todo lo que necesita** para correr igual en cualquier máquina. Se acaba el "en mi máquina sí funciona".
📌 `Dockerfile` en la raíz del proyecto.

## **Imagen vs contenedor**
**Imagen** = la *plantilla* (como un instalador): archivos + configuración, no cambia.
**Contenedor** = una imagen *corriendo* (como el programa abierto). De una imagen salen muchos contenedores.
📌 `orders-api:dev` es la imagen; cada `docker run` crea un contenedor.

## **Dockerfile multistage**
Varias etapas en un archivo: una **compila** (imagen grande con Go) y otra **solo corre** el binario (imagen mínima).
📌 Etapa `build` (~300 MB) → imagen final distroless (~36 MB).

## **Caché de capas**
Cada instrucción del Dockerfile es una capa guardada. Si no cambió, Docker la reutiliza.
📌 Se copia `go.mod` antes que el código: si solo cambias código, no se vuelven a descargar dependencias.

## **Distroless / no root**
Imagen sin shell ni herramientas: menos cosas que un atacante pueda usar. Y el proceso corre con un **usuario sin privilegios**.
📌 `gcr.io/distroless/static` + `USER 65532`.

## **Kubernetes (K8s)**
Orquestador: corre contenedores en varias máquinas, los **reinicia si fallan**, los **escala** y los **actualiza sin cortar el servicio**.

## **Manifiesto**
Archivo YAML donde **declaras cómo quieres** que esté todo. Kubernetes se encarga de que la realidad coincida.
📌 Carpeta `deploy/k8s/`.

## **Pod**
La unidad mínima de Kubernetes: **uno o más contenedores** que corren juntos. Es temporal: muere y se reemplaza con otra IP.

## **Deployment**
"Quiero **N copias** de este pod siempre". Si uno muere, crea otro. Maneja las **actualizaciones**.
📌 `api.yaml`: Deployment `orders-api` con `replicas: 1`.

## **Service**
Una **dirección fija** (nombre DNS) para llegar a los pods, aunque cambien de IP. Reparte el tráfico entre los pods **listos**.
📌 La API llama al catálogo como `http://catalog`: ese es el Service.

## **ConfigMap**
Configuración **no sensible** fuera de la imagen. La misma imagen sirve para dev, QA y producción.
📌 `configmap.yaml`: `PORT`, `CATALOG_URL`, `SHUTDOWN_DELAY`.

## **Secret**
Igual que el ConfigMap pero para datos **sensibles** (contraseñas, tokens). ⚠️ Por defecto solo está en **base64**, que **no es cifrado**: cualquiera con acceso al clúster puede leerlo.

## **Secret de K8s vs gestión completa de secretos**
Un Secret de K8s solo **guarda** el dato. Una gestión completa (Vault, AWS Secrets Manager, Azure Key Vault) además: **cifra**, **controla quién accede**, **registra quién lo leyó**, **rota** las claves automáticamente y nunca deja el secreto en Git.

## **Liveness probe (`/healthz`)**
"¿Estás **vivo**?" Si falla → Kubernetes **reinicia** el pod.
⚠️ **No** debe revisar dependencias: si el catálogo cae, reiniciar la API no arregla nada.
📌 `health.go`: siempre responde 200 si el proceso funciona.

## **Readiness probe (`/readyz`)**
"¿Puedes **atender** tráfico?" Si falla → Kubernetes **deja de enviarle solicitudes**, pero **no lo reinicia**.
📌 Responde 503 mientras la API arranca y cuando se está apagando.

## **Liveness vs readiness (pregunta clave)**
Liveness = **reiniciar** si está trabado. Readiness = **sacar del tráfico** si no puede atender por ahora.

## **Graceful shutdown + drenado**
Al apagarse: 1) `/readyz` pasa a 503, 2) espera unos segundos a que K8s deje de mandarle tráfico, 3) termina lo que estaba atendiendo, 4) sale.
📌 `SHUTDOWN_DELAY` en `main.go`.

## **Requests y limits**
**Requests** = recursos **garantizados**; Kubernetes los usa para decidir en qué máquina cabe el pod.
**Limits** = el **máximo**: si pasa la memoria lo matan (*OOMKilled*); si pasa la CPU lo frenan.
📌 API: request 50m CPU / 32Mi, limit 500m / 128Mi.

## **Rolling update**
Actualizar **de a poco**: crea un pod nuevo, espera que esté *ready*, y recién ahí apaga uno viejo. **Cero caída.**
📌 `maxSurge: 1`, `maxUnavailable: 0`.

## **Rollback**
Volver a la versión anterior si la nueva falla: `kubectl rollout undo deployment/orders-api`.

## **HPA (Horizontal Pod Autoscaler)**
Sube o baja **automáticamente** la cantidad de pods según la carga (por ejemplo, CPU > 70%). Necesita un *metrics-server* y que los pods tengan **requests** definidos.

## **Escalado horizontal y estado**
Solo se puede escalar a varios pods si **no guardan datos en memoria**.
📌 La API tiene `replicas: 1` porque guarda pedidos en memoria. El catálogo tiene `replicas: 2` porque no guarda estado. Para escalar la API → PostgreSQL.

## **kind**
"Kubernetes IN Docker": un clúster de Kubernetes **local**, dentro de Docker, para practicar.

## **¿Qué problemas siguen después de contenerizar?**
Docker **no** resuelve: datos persistentes (bases de datos), secretos, observabilidad (logs, métricas), seguridad de la imagen, ni el diseño de la app (si guarda estado en memoria, no escala).

---

## 🎯 **Frase para la entrevista**
> "La API corre en una imagen multistage distroless sin root. En Kubernetes uso un Deployment con rolling update sin caída, un Service para llegar a los pods y un ConfigMap para la configuración. Separo liveness, que no revisa dependencias para no provocar reinicios en cascada, de readiness, que saca al pod del tráfico mientras arranca o se apaga. Defino requests y limits, y dejo una réplica porque el estado está en memoria; para escalar con HPA primero movería los datos a una base de datos."
