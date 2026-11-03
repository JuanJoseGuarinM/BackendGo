# Guía de conceptos (para estudiar y explicar en entrevista)

Cada concepto tiene: **qué es**, **dónde está en el proyecto** y **cómo explicarlo en una entrevista**.
Se va llenando a medida que avanzamos.

---

## Jornada 1 — Fundamentos de Go

Demo de toda la jornada: `go run ./cmd/practice` · Pruebas: `go test ./...`

### Módulo y paquete
- **Qué es:** un *paquete* es una carpeta con archivos `.go`. Un *módulo* es el proyecto completo (muchos paquetes), definido por `go.mod`.
- **En el proyecto:** [go.mod](../go.mod) define el módulo `orders-api`; cada carpeta de `internal/practice/` es un paquete.
- **En entrevista:** "El módulo es la unidad de versionado y dependencias; el paquete es la unidad de organización. Uso `cmd/` para los ejecutables e `internal/` para el código que no quiero que otros módulos importen; eso lo hace cumplir el compilador."

### Errores como valores
- **Qué es:** en Go no hay `try/catch`. Una función que puede fallar devuelve el resultado **y** un `error`. Quien la llama revisa `if err != nil`.
- **En el proyecto:** `Reserve` devuelve `(int, error)` en [inventory.go](../internal/practice/inventory/inventory.go).
- **En entrevista:** "En Go los errores son valores explícitos en la firma. Obliga a manejarlos donde ocurren y deja visible qué puede fallar."

### Error centinela (sentinel)
- **Qué es:** un error con nombre, creado una vez, para reconocerlo después. `var ErrProductNotFound = errors.New("...")`.
- **En el proyecto:** `ErrProductNotFound`, `ErrInsufficientStock` en [inventory.go](../internal/practice/inventory/inventory.go).
- **En entrevista:** "Uso centinelas para fallos conocidos del negocio; la capa HTTP pregunta con `errors.Is` qué pasó y elige el código (404, 409...) sin comparar textos."

### Error con datos (tipo propio)
- **Qué es:** un struct con método `Error() string`. Sirve cuando además de *qué* pasó necesitas *datos* (qué campo falló).
- **En el proyecto:** `ValidationError` en [inventory.go](../internal/practice/inventory/inventory.go).
- **En entrevista:** "Si el que maneja el error necesita información extra, creo un tipo de error y lo extraigo con `errors.As`."

### Envolver errores con `%w` (wrapping)
- **Qué es:** agregar contexto a un error sin perder el original: `fmt.Errorf("reservar %q: %w", sku, err)`.
- **En el proyecto:** `Reserve` en [inventory.go](../internal/practice/inventory/inventory.go) y `Complete` en [checkout.go](../internal/practice/checkout/checkout.go).
- **En entrevista:** "Envuelvo con `%w` para que el mensaje diga dónde falló sin perder el error original; con `%v` solo copiaría el texto y `errors.Is` ya no lo encontraría."

### `errors.Is` y `errors.As`
- **Qué es:** `Is` = "¿este error es (o contiene) tal centinela?". `As` = "¿hay un error de tal tipo? dámelo para leer sus campos". Ambos revisan toda la cadena de errores envueltos.
- **En el proyecto:** [cmd/practice/errors.go](../cmd/practice/errors.go).
- **En entrevista:** "`Is` para centinelas, `As` para errores con datos. Funcionan aunque el error venga envuelto varias veces."

### Error vs panic
- **Qué es:** `error` = fallo **esperable** (dato inválido, servicio caído). `panic` = error **del programador** o estado imposible; detiene el programa.
- **En el proyecto:** no usamos `panic` en ningún lado, a propósito.
- **En entrevista:** "Uso `error` para todo lo que puede pasar en operación normal. `panic` solo para bugs irrecuperables; nunca para controlar el flujo. En un servidor HTTP, Gin tiene un middleware de *recovery* para que un panic no tumbe el proceso."

### Interfaces
- **Qué es:** un conjunto de métodos. Un tipo la cumple **de forma implícita** si tiene esos métodos (no se escribe `implements`).
- **En el proyecto:** `Notifier` en [checkout.go](../internal/practice/checkout/checkout.go). `ConsoleNotifier` y `*MemoryNotifier` en [notify.go](../internal/practice/notify/notify.go) la cumplen sin mencionarla.
- **Criterios para definir una interfaz:**
  1. La define **quien la usa** (consumidor), no quien la implementa.
  2. **Pequeña**: solo los métodos que el consumidor necesita.
  3. Se crea cuando hay una razón: varias implementaciones o poder simular en pruebas.
- **En entrevista:** "Defino interfaces pequeñas del lado del consumidor. Así `checkout` no depende de cómo se envían los mensajes, y en pruebas le paso un falso (`failingNotifier`) para simular fallos."

### Inyección de dependencias
- **Qué es:** pasarle a un componente lo que necesita desde afuera, en vez de que él lo cree.
- **En el proyecto:** `NewService(n Notifier)` en [checkout.go](../internal/practice/checkout/checkout.go); [cmd/practice/interfaces.go](../cmd/practice/interfaces.go) decide qué Notifier usar.
- **En entrevista:** "El `main` arma las piezas y las inyecta. Los componentes reciben interfaces, lo que los hace fáciles de probar y cambiar."

### Receptor por valor vs por puntero
- **Qué es:** `func (c T) M()` trabaja sobre una **copia**; `func (c *T) M()` trabaja sobre el **original**.
- **Cuándo puntero:** si el método modifica el struct, si el struct es grande, o si contiene un `sync.Mutex` (no se debe copiar).
- **En el proyecto:** `ConsoleNotifier` (valor, no modifica nada) vs `*MemoryNotifier` (puntero, agrega mensajes y tiene mutex) en [notify.go](../internal/practice/notify/notify.go).
- **Detalle:** si el método tiene receptor puntero, solo `*T` cumple la interfaz. Lo verificamos en compilación en [checkout_test.go](../internal/practice/checkout/checkout_test.go).
- **En entrevista:** "Uso puntero cuando el método muta el estado o el tipo tiene un mutex; valor para tipos pequeños e inmutables. Y mantengo el mismo tipo de receptor en todos los métodos de un tipo."

### Composición (embedding)
- **Qué es:** Go no tiene herencia. Se **incrusta** un tipo dentro de otro y sus métodos se "promueven".
- **En el proyecto:** `PrefixNotifier` incrusta `*MemoryNotifier`: redefine `Notify` y hereda `Sent()` en [notify.go](../internal/practice/notify/notify.go).
- **En entrevista:** "Go favorece la composición sobre la herencia: incrusto un tipo para reutilizar su comportamiento y sobrescribo solo lo que cambia."

### `context.Context`
- **Qué es:** un objeto que viaja de función en función llevando **cancelación, tiempo límite (deadline)** y valores de la solicitud.
- **Reglas:** va como **primer parámetro** (`ctx`), se **propaga** a todo lo que llamas, y si creas uno con `WithTimeout`/`WithCancel` siempre haces `defer cancel()`.
- **En el proyecto:** `RunAll` y `SleepTask` en [concurrency.go](../internal/practice/concurrency/concurrency.go); `Complete` lo pasa a `Notify` en [checkout.go](../internal/practice/checkout/checkout.go).
- **En entrevista:** "Propago el context desde la solicitud HTTP hasta la última llamada. Si el cliente se desconecta o vence el timeout, todo lo que está haciendo trabajo para esa solicitud se entera y se detiene."

### Goroutines
- **Qué es:** funciones que corren **en paralelo**, muy livianas (miles sin problema). Se lanzan con `go f()` o `wg.Go(f)`.
- **En el proyecto:** `RunAll` lanza una goroutine por tarea en [concurrency.go](../internal/practice/concurrency/concurrency.go). En la demo, dos tareas de 300ms y 200ms tardan **300ms**, no 500ms.
- **Goroutine leak:** una goroutine que queda bloqueada para siempre. `RunAll` lo evita: canal con buffer + espera a todas (`wg.Wait`) antes de volver.
- **En entrevista:** "Toda goroutine que lanzo debe tener una forma clara de terminar: un context que la cancele y alguien que espere su fin."

### Channels
- **Qué es:** "tubos" para pasar datos entre goroutines de forma segura. `ch <- v` envía, `v := <-ch` recibe.
- **`select`:** espera varios canales y reacciona al **primero** que esté listo.
- **En el proyecto:** canal `results` en `RunAll`; `select` entre `timer.C` y `ctx.Done()` en `SleepTask` ([concurrency.go](../internal/practice/concurrency/concurrency.go)).
- **En entrevista:** "Uso channels para pasar resultados entre goroutines y `select` con `ctx.Done()` para que una tarea se detenga apenas la cancelan."

### Mutex y condición de carrera
- **Condición de carrera (race condition):** dos goroutines usan la misma variable a la vez y al menos una escribe. Resultado impredecible (ej. `n++` pierde incrementos).
- **Mutex:** candado. `Lock()` / `defer Unlock()`: solo una goroutine a la vez entra a la sección protegida.
- **`go test -race`:** detecta condiciones de carrera al ejecutar las pruebas.
- **En el proyecto:** `Counter` en [concurrency.go](../internal/practice/concurrency/concurrency.go) y `MemoryNotifier` en [notify.go](../internal/practice/notify/notify.go). Pruebas con 1000 y 100 goroutines.
- **En entrevista:** "Si varias goroutines comparten un estado, lo protejo con mutex y lo verifico con `-race` en CI."

### Channels vs memoria compartida (pregunta de sustentación)
- **En entrevista:** "Son dos estilos. Con **channels** las goroutines se pasan los datos y nadie los comparte: ideal para coordinar tareas y pasar resultados (`RunAll`). Con **mutex** varias goroutines comparten una variable protegida: más simple para un estado pequeño como un contador o un caché (`Counter`). El lema de Go es *'comparte memoria comunicándote'*, pero uso el que deje el código más simple."

### ¿Cuándo la concurrencia aporta y cuándo solo complica?
- **Aporta:** tareas **independientes** que **esperan** algo externo (llamar a 2 servicios, leer varios archivos). En la demo: 300ms en vez de 500ms.
- **Complica:** tareas que dependen una de otra, trabajo muy corto, o cuando agrega estado compartido sin necesidad. Trae riesgo de carreras, leaks y código más difícil de probar.
- **En entrevista:** "Agrego concurrencia cuando hay esperas independientes que puedo solapar y lo puedo medir; si no, el código secuencial es más simple y correcto."

### Prueba table-driven
- **Qué es:** los casos de prueba se escriben como una **tabla** (lista de structs con entrada y resultado esperado) y un solo `for` los ejecuta con `t.Run`.
- **En el proyecto:** [inventory_test.go](../internal/practice/inventory/inventory_test.go), [checkout_test.go](../internal/practice/checkout/checkout_test.go).
- **En entrevista:** "Es el estilo idiomático en Go: agregar un caso es agregar una fila, y cada caso aparece con su nombre si falla."

### Doble de prueba (fake)
- **Qué es:** una implementación falsa de una interfaz, solo para pruebas.
- **En el proyecto:** `failingNotifier` en [checkout_test.go](../internal/practice/checkout/checkout_test.go) simula que el correo está caído.
- **En entrevista:** "Gracias a las interfaces puedo simular fallos externos en pruebas sin levantar servicios reales."
