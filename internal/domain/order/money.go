package order

import "errors"

// ErrNegativeMoney indica que se intentó crear un monto negativo.
var ErrNegativeMoney = errors.New("el monto no puede ser negativo")

// Money es un VALUE OBJECT (objeto de valor).
//
// Características de un Value Object:
//   - Se define por su VALOR, no por un ID: dos Money de 1500 son iguales.
//   - Es INMUTABLE: Add y Times devuelven un Money nuevo, no modifican el actual.
//   - Protege una INVARIANTE: no existe un Money negativo. El campo es privado
//     y la única forma de crearlo es NewMoney, que lo valida.
//
// Se guarda en centavos (int64) porque float64 tiene errores de redondeo con
// dinero (0.1 + 0.2 != 0.3).
type Money struct {
	cents int64
}

// NewMoney es la única puerta de entrada para crear un Money válido.
func NewMoney(cents int64) (Money, error) {
	if cents < 0 {
		return Money{}, ErrNegativeMoney
	}
	return Money{cents: cents}, nil
}

// Cents devuelve el monto en centavos.
func (m Money) Cents() int64 { return m.cents }

// Add suma dos montos. Receptor por valor: Money es pequeño e inmutable.
func (m Money) Add(other Money) Money {
	return Money{cents: m.cents + other.cents}
}

// Times multiplica el monto por una cantidad.
//
// Si n es negativo es un error DEL PROGRAMADOR (OrderItem ya valida la
// cantidad), no un error esperable del usuario. Por eso panic y no error:
// es el caso de uso correcto de panic que vimos en la J1.
func (m Money) Times(n int) Money {
	if n < 0 {
		panic("order.Money.Times: n no puede ser negativo")
	}
	return Money{cents: m.cents * int64(n)}
}
