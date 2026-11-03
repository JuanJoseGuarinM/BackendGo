# Jornada 3 — DDD (resumen rápido)

---

## **DDD (Domain-Driven Design)**
Diseñar el código alrededor del negocio: mismas palabras y reglas que usa el negocio, concentradas en el dominio.
📌 `internal/domain/order` habla de *pedido*, *confirmar*, *monto*.

## **¿Por qué las carpetas no son DDD?**
Tener una carpeta `domain/` no sirve si cualquiera puede cambiar los datos y saltarse las reglas. DDD es que **el modelo proteja sus reglas**.
📌 En la J2 se podía escribir `o.Status = "CONFIRMED"`. En la J3 los campos son privados y eso no compila.

## **Lenguaje ubicuo**
El mismo vocabulario en el negocio y en el código.
📌 El negocio dice "confirmar pedido" → el método se llama `Confirm()`.

## **Entidad**
Objeto con **ID propio**. Aunque cambien sus datos, sigue siendo el mismo.
📌 `Order`: pasa de PENDING a CONFIRMED y sigue siendo el mismo pedido.

## **Value Object**
Objeto **sin ID** que vale por su valor. **No cambia** (inmutable) y **se valida al crearse**. Solo se crea si protege una regla.
📌 `Money`: nunca negativo; dos Money de 1500 son iguales.

## **Agregado**
Grupo de objetos que se cambian y se guardan **como una sola unidad**. Desde afuera solo se usa la **raíz**.
📌 `Order` (raíz) + sus `OrderItem`. Los items solo cambian a través de `Order`.

## **Invariante**
Regla que **siempre** debe cumplirse.
📌 Pedido con cliente · monto no negativo · no confirmar vacío · confirmado no cambia.

## **Encapsulamiento**
Campos privados (en minúscula) + métodos que validan antes de cambiar.
📌 `Order{id, status...}` son privados; se leen con `ID()`, `Status()`.

## **`New` vs `Restore`**
`New` crea un pedido nuevo y valida. `Restore` reconstruye uno que ya existía (desde la BD).

## **Repositorio**
Guarda y carga **agregados completos**, sin que el resto sepa si es memoria o PostgreSQL.
📌 La interfaz `OrderRepository` está en `application` (quien la usa).

## **Caso de uso**
Coordina los pasos: buscar → regla del dominio → guardar. **No tiene reglas propias.**
📌 `Confirm` en `order_service.go`: si la regla falla, no guarda.

## **Error de transporte vs error de dominio**
- Transporte = **mal escrito** (JSON roto) → lo detecta HTTP → **400**.
- Dominio = **bien escrito pero viola una regla** (pedido vacío) → lo detecta el dominio → **409/422**.

## **Ubicación de interfaces**
La interfaz vive con **quien la usa**, no con quien la implementa.

## **DDD vs Hexagonal**
DDD = **cómo modelar** el negocio. Hexagonal = **cómo aislarlo** de HTTP y la BD. Se complementan.

## **Bounded context**
Límite donde una palabra significa una sola cosa.
📌 "Producto" en Ventas (precio) ≠ "producto" en Bodega (peso).

## **Transacción y concurrencia**
Una transacción = un agregado. Para que dos `confirm` simultáneos no se pisen: **bloqueo optimista** (una columna `version`).

---

## 🎯 **Frase para la entrevista**
> "Con DDD modelo el negocio en el dominio y protejo sus invariantes con campos privados, constructores y métodos que validan. `Order` es mi agregado, `Money` mi value object, y el repositorio guarda el agregado completo detrás de una interfaz definida por la aplicación."
