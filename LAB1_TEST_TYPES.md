# Где какие виды тестов

## Классический стиль (без mock/stub)
- `code/components/business/internal/usecase/user_usecase_test.go` (in‑memory репозиторий)
- `code/components/business/internal/usecase/auth_usecase_test.go` (in‑memory репозитории)
- `code/components/data/internal/repository/localjson/*_test.go` (реальные localjson репозитории)

## Лондонский стиль (через моки)
- `code/components/business/internal/usecase/game_usecase_test.go`
- `code/components/business/internal/usecase/move_usecase_test.go`
- `code/components/business/internal/usecase/user_usecase_test.go` (моки репозитория)

## Тесты на ошибки/исключения
- `code/components/business/internal/usecase/animate_move_test.go`
- `code/components/business/internal/usecase/game_usecase_test.go`
- `code/components/business/internal/usecase/move_usecase_test.go`
- `code/components/business/internal/usecase/user_usecase_test.go`
- `code/components/business/internal/usecase/auth_usecase_test.go`
- `code/components/data/internal/repository/localjson/*_test.go`

## Fixture (фикстуры)
- `code/components/data/internal/repository/localjson/test_helpers_test.go`

## Data Builder
- `code/components/business/internal/usecase/test_helpers_test.go`
- `code/components/data/internal/repository/localjson/test_helpers_test.go`

## Object Mother (Fabric/Object Mother)
- `code/components/business/internal/usecase/test_helpers_test.go`
- `code/components/data/internal/repository/localjson/test_helpers_test.go`
