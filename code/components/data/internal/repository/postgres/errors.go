package postgres

import sdkrepo "ppo/sdk/port/repo"

var (
	ErrNotFound      = sdkrepo.ErrNotFound
	ErrAlreadyExists = sdkrepo.ErrAlreadyExists
	ErrInvalidData   = sdkrepo.ErrInvalidData
)
