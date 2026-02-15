package domain

type Login string

type PasswordHash string

type UserCredentials struct {
	UserID       UserID
	Login        Login
	PasswordHash PasswordHash
}
