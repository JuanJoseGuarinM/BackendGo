# Jornada 7 — Arquitecturas con IA (resumen rápido)

---

## **¿Cuándo la IA aporta valor?**
Cuando el problema es **ambiguo** o usa **lenguaje natural** (clasificar reclamos, resumir, buscar por significado) y un error ocasional es **aceptable y reversible**.
📌 Clasificar pedidos: si se equivoca, no pasa nada grave.

## **¿Cuándo es mejor una regla fija (determinista)?**
Cuando la lógica **se puede escribir** y debe ser **exacta, explicable y auditable**: precios, montos, permisos, reglas del negocio.
📌 "No confirmar un pedido vacío" es una regla, nunca IA. `Rules` en `classifier/rules.go`.

## **Aislar la IA del dominio**
La IA va detrás de una **interfaz** (`OrderClassifier`), como cualquier dependencia externa. El dominio no sabe que existe.
📌 Cambiar de proveedor = cambiar un adaptador.

## **Degradación controlada (fallback)**
Si la IA falla, tarda o responde mal, el sistema **sigue funcionando** con un plan B.
📌 Reglas fijas de respaldo → siempre 200, con `source: "rules"`.

## **Inferencia**
Usar un modelo ya entrenado para obtener una respuesta: le mandas datos, te devuelve una predicción.
📌 `Simulated.Classify` simula una inferencia.

## **Confianza (confidence)**
Qué tan seguro está el modelo (0 a 1). **No es certeza.** Por debajo de un umbral, mejor no confiar.
📌 Umbral 0.6: con 16 unidades el modelo dice 0.55 → se usan las reglas.

## **Alucinación**
El modelo inventa una respuesta que **suena bien pero es falsa o inválida**.
📌 Modo `invalid`: responde `URGENTE_VIP`, una etiqueta que no existe → se rechaza.

## **Evaluación / validación de respuestas**
Nunca usar la salida del modelo sin revisarla: **lista de valores permitidos**, rangos, formato.
📌 `askModel` en `classification.go` valida etiqueta y confianza.

## **LLM (modelo de lenguaje)**
Modelo que entiende y genera texto. Llamarlo = **una pregunta, una respuesta**.

## **Embeddings**
Convertir un texto en una **lista de números** que representa su significado. Textos parecidos quedan "cerca". Sirve para **buscar por significado**, no por palabras exactas.

## **RAG (Retrieval-Augmented Generation)**
Antes de preguntarle al LLM, **buscas documentos relevantes** (con embeddings) y se los pasas como contexto. Así responde con **tu información** y alucina menos.
📌 Ejemplo: responder preguntas sobre las políticas de devolución de la empresa.

## **Agente**
El modelo **decide qué herramientas usar** (buscar, consultar una API, ejecutar acciones) en **varios pasos** hasta cumplir un objetivo. Más poderoso, pero más riesgoso.

## **Herramientas (tools)**
Funciones que un agente puede llamar: "consultar pedido", "enviar correo". Cada una necesita **permisos mínimos** y, si cambia algo, **aprobación**.

## **LLM vs RAG vs agente (pregunta clave)**
**LLM** = pregunta → respuesta. **RAG** = busca contexto → pregunta → respuesta. **Agente** = el modelo planifica y llama herramientas en varios pasos.

## **Riesgo: privacidad**
Lo que envías a un proveedor externo sale de tu control. **Minimización de datos:** solo lo necesario.
📌 `ClassificationInput` no tiene el cliente: es imposible enviarlo por error.

## **Riesgo: latencia**
Los modelos pueden tardar segundos. **Timeout** siempre y, si se puede, fuera del camino crítico.
📌 Timeout de 1 s; endpoint aparte: crear o confirmar un pedido nunca espera a la IA.

## **Riesgo: costo**
Cada llamada cuesta dinero (por tokens). Llamar solo cuando aporta, cachear resultados, usar modelos más baratos cuando alcanza.

## **Riesgo: fallos**
El proveedor puede caerse o limitarte. **Fallback** y nunca un 5xx por culpa del modelo.

## **Riesgo: credenciales**
La API key del proveedor va en un **gestor de secretos**, nunca en el código ni en Git. Rotarla y limitar sus permisos.

## **Observabilidad de la IA**
Registrar cada llamada: **latencia, etiqueta, confianza, si falló y de dónde salió la respuesta** (modelo o reglas).
📌 Log `llamada al clasificador IA` + campo `source` en la respuesta.

## **Transparencia**
Decir **de dónde salió** una decisión y **por qué**.
📌 La respuesta incluye `source` y `reason`.

---

## 🎯 **Frase para la entrevista**
> "Aislé la IA detrás de una interfaz `OrderClassifier`, en un endpoint aparte para que ninguna operación del negocio dependa de ella. Le pongo timeout, valido su respuesta contra etiquetas permitidas y un umbral de confianza, y si falla, tarda o alucina uso reglas deterministas, siempre indicando la fuente. Al modelo solo le envío los datos necesarios. Las reglas exactas, como no confirmar un pedido vacío, nunca las dejo en manos de la IA."
