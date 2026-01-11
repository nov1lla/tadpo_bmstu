package component

import (
	"time"

	"ppo/sdk/port/repo"
	"ppo/sdk/usecase"
)

// DataProvider предоставляет доступ к инфраструктурным реализациям репозиториев.
type DataProvider interface {
	UserRepository() repo.UserRepository
	GameRepository() repo.GameRepository
	MoveRepository() repo.MoveRepository
	UserCredentialsRepository() repo.UserCredentialsRepository
	Close() error
}

// BusinessProvider раскрывает собранные экземпляры юзкейсов приложения.
type BusinessProvider interface {
	UserUseCase() usecase.UserUseCase
	GameUseCase() usecase.GameUseCase
	MoveUseCase() usecase.MoveUseCase
	AuthUseCase() usecase.AuthUseCase
	MoveAnimationUseCase() usecase.MoveAnimationUseCase
	Close() error
}

// BusinessConfig описывает параметры, необходимые для настройки слоя бизнес-логики.
type BusinessConfig struct {
	OpenAIKey         string
	OpenAIModel       string
	OpenAIBaseURL     string
	OpenAITemperature float64
	HTTPTimeout       time.Duration
}
