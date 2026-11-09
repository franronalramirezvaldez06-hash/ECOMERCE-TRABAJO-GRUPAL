// Package apperr define los errores de negocio compartidos por todos los
// modulos. La capa HTTP los traduce a codigos de estado.
package apperr

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound          = errors.New("recurso no encontrado")
	ErrConflict          = errors.New("conflicto con el estado actual")
	ErrInsufficientStock = errors.New("stock insuficiente")
	ErrUnauthorized      = errors.New("no autenticado")
	ErrForbidden         = errors.New("no tienes permisos para esta accion")
	ErrGateway           = errors.New("error en la pasarela de pago")
)

// ValidationError indica datos de entrada invalidos.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

// Validation crea un ValidationError con formato.
func Validation(format string, args ...any) error {
	return &ValidationError{Msg: fmt.Sprintf(format, args...)}
}
