# Jornada 4 — Integración y decisiones de arquitectura (resumen rápido)

---

## **Integración entre servicios**
Cuando tu servicio necesita datos de **otro servicio** por la red.
📌 Al crear un pedido, la API le pregunta el precio al **catálogo** (`cmd/catalog`).

## **Comunicación síncrona**
Llamas y **esperas la respuesta** para seguir (HTTP, gRPC). Simple, pero si el otro está caído, tú también fallas.
📌 `GetProduct` en `Create`: sin precio no hay pedido.

## **Comunicación asíncrona**
Envías un **mensaje** a una cola y **sigues sin esperar**. El otro lo procesa cuando pueda. Más resistente a caídas, pero más complejo.
📌 Ejemplo: publicar "pedido creado" para que *Notificaciones* envíe un correo más tarde.

## **Acoplamiento**
Qué tanto depende un servicio de otro. **Síncrono = más acoplado** (si el catálogo cae, no se crean pedidos).
📌 Lo reducimos con una **interfaz** (`CatalogClient`): la aplicación no sabe si es HTTP o memoria.

## **Contrato**
El acuerdo de **cómo se hablan** dos servicios: URL, campos del JSON, códigos de respuesta.
📌 `GET /products/{sku}` → `{"sku","name","price_cents"}` (`productDTO` en `http_client.go`).

## **Timeout**
Tiempo **máximo** que esperas una respuesta. Sin timeout, un servicio lento te deja colgado.
📌 `AttemptTimeout: 1s` por intento. Catálogo de 5 s → respondemos en ~3 s, sin colgarnos.

## **Retry (reintento)**
Volver a intentar cuando la falla es **temporal** (5xx, red caída, timeout). **No** se reintenta un 404: la respuesta no va a cambiar.
📌 `MaxRetries: 2` → 3 intentos en total.

## **Backoff exponencial + jitter**
Esperar **cada vez más** entre reintentos (100 ms, 200 ms, 400 ms) + un poco de **azar**. Así no saturas a un servicio que se está recuperando.
📌 función `backoff` en `http_client.go`.

## **Idempotencia y reintentos**
Solo se reintenta lo **seguro e idempotente** (GET). Reintentar un POST que **cobra** o **crea** puede duplicarlo.
📌 Reintentamos `GET /products`, pero nunca reintentaríamos un "cobrar pedido" sin una *idempotency key*.

## **Circuit breaker**
Como el **breaker de la luz**: si el servicio falla muchas veces seguidas, **deja de llamarlo** por un rato y falla rápido. Luego prueba de nuevo. Protege a los dos servicios.
📌 No implementado; sería el siguiente paso si el catálogo fallara seguido.

## **Falla parcial**
En sistemas distribuidos, **una parte falla y el resto sigue**. Hay que decidir qué hacer en vez de caerse completo.
📌 Catálogo caído → la API responde **503** controlado; consultar y confirmar pedidos sigue funcionando.

## **503 Service Unavailable**
"Una dependencia no está disponible, intenta más tarde". No es culpa del cliente (4xx) ni un bug nuestro (500).

## **Consistencia eventual**
Los datos de distintos servicios **no coinciden al instante**, pero **terminan coincidiendo**. Es el precio de lo asíncrono.
📌 Si *Inventario* se entera del pedido por un evento, por unos segundos su stock "no sabe" de él.

## **REST (HTTP + JSON)**
Recursos con URL y métodos HTTP. Simple, universal y fácil de probar con el navegador o curl.
📌 Lo usamos en la API y en el catálogo.

## **SOAP + WSDL**
Protocolo antiguo basado en **XML**. El **WSDL** es un archivo que describe el contrato exacto (operaciones y tipos). Pesado pero muy estricto.
📌 Se usa al integrar con **sistemas legados**: bancos, ERP, entidades del gobierno.

## **gRPC + Protocol Buffers**
Llamadas a funciones remotas, **binario y muy rápido** sobre HTTP/2. El contrato se escribe en un archivo `.proto` y se genera el código.
📌 Ideal **entre microservicios internos** con mucho tráfico. Menos práctico para navegadores.

## **Cola (queue)**
Un mensaje lo procesa **un solo** consumidor. Reparte el trabajo.
📌 "Generar factura del pedido X": lo hace un solo worker.

## **Pub/Sub (publicar/suscribir)**
Un mensaje (evento) lo reciben **todos los suscriptores**.
📌 "Pedido creado" → lo reciben *Notificaciones*, *Inventario* y *Analítica*.

## **Evento**
Un mensaje que dice **algo que ya pasó**, en pasado: `OrderCreated`, `OrderConfirmed`.

## **Consumidor idempotente**
Las colas pueden entregar **el mismo mensaje dos veces**. El consumidor guarda los IDs ya procesados y, si llega repetido, lo ignora.

## **Anticorruption layer (traducción en el adaptador)**
El adaptador **traduce** el modelo del otro servicio al nuestro, para que su formato no se meta en nuestro código.
📌 `productDTO` (del catálogo) → `application.Product` (nuestro).

## **No confiar en el cliente**
Datos sensibles como el **precio** los decide el servidor, nunca quien envía el request.
📌 Ya no se acepta `unit_price` en el JSON: el precio viene del catálogo.

---

## 🎯 **Frase para la entrevista**
> "Integré el catálogo por HTTP detrás de una interfaz `CatalogClient`. Cada llamada usa el context, con timeout por intento y reintentos con backoff exponencial solo para fallas temporales, porque el GET es idempotente. Si el catálogo cae, respondo 503 controlado. Elegí HTTP por simplicidad; usaría gRPC entre servicios internos de alto tráfico, SOAP para sistemas legados y mensajería asíncrona cuando no necesito la respuesta al instante."
