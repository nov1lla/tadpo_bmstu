package repo

import "errors"

var (
	ErrNotFound      = errors.New("entity not found")
	ErrAlreadyExists = errors.New("entity already exists")
	ErrInvalidData   = errors.New("entity data is invalid")
)
