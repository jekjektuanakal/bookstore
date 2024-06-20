package store

import "fmt"

var (
	ErrConflict     = fmt.Errorf("conflict")
	ErrNotFound     = fmt.Errorf("not found")
	ErrInvalid      = fmt.Errorf("invalid")
	ErrUnauthorized = fmt.Errorf("unauthorized")
	ErrInternal     = fmt.Errorf("internal error")
)
