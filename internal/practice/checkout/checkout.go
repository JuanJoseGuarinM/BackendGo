// Package checkout muestra cómo y DÓNDE definir una interfaz.
//
// Regla: la interfaz la define quien la USA (el consumidor), no quien la
// implementa. checkout necesita "algo que envíe mensajes" y lo declara aquí,
// con solo el método que necesita. Así checkout no depende de notify, y en las
// pruebas podemos pasarle un falso. Es la misma idea que usaremos en la
// Jornada 3 con el repositorio de pedidos.
package checkout

import (
	"context"
	"errors"
	"fmt"
)

// Notifier es lo único que checkout necesita del exterior.
// Interfaz pequeña (un método): fácil de implementar y de simular.
type Notifier interface {
	Notify(ctx context.Context, to, body string) error
}

// ErrEmptyCustomer se devuelve cuando no se indica el cliente.
var ErrEmptyCustomer = errors.New("cliente vacío")

// Service completa compras. Depende de la interfaz, no de un tipo concreto.
type Service struct {
	notifier Notifier
}

// NewService recibe sus dependencias desde afuera (inyección de dependencias).
// Quien arma el programa (main) decide qué Notifier concreto usar.
func NewService(n Notifier) *Service {
	return &Service{notifier: n}
}

// Complete confirma la compra y avisa al cliente.
// El ctx se recibe como primer parámetro y se PROPAGA a lo que llamamos: si la
// solicitud se cancela, la notificación también se entera.
func (s *Service) Complete(ctx context.Context, customer string) error {
	if customer == "" {
		return ErrEmptyCustomer
	}
	if err := s.notifier.Notify(ctx, customer, "tu pedido fue confirmado"); err != nil {
		return fmt.Errorf("completar compra de %q: %w", customer, err)
	}
	return nil
}
