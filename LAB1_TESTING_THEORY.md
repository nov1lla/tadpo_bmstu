# Лабораторная 1: теория и примеры из проекта

Ниже кратко объяснены требования и практики тестирования, с примерами из нашего кода.

## Unit‑тесты (модульные)
Идея: проверяем отдельный класс/структуру без запуска всего приложения.

Примеры:
- `code/components/business/internal/usecase/user_usecase_test.go` — тесты логики `UserUseCase`.
- `code/components/data/internal/repository/localjson/user_repository_test.go` — тесты репозитория пользователя.

## Arrange‑Act‑Assert (AAA)
Структура теста:
1) Arrange — подготовка данных и зависимостей.
2) Act — один вызов метода.
3) Assert — проверка результата.

Пример (AAA):
- `TestUserUseCaseCreateAndGet_Classic` в `code/components/business/internal/usecase/user_usecase_test.go`.
  - Arrange: создаем in‑memory репозиторий и use case.
  - Act: вызываем `Create`, затем `Get`.
  - Assert: проверяем ID и корректность загрузки.

## Классический vs “Лондонский” стиль
Классический стиль:
- Минимум моков, тестируем реальное состояние/данные (in‑memory).
- Пример: `TestUserUseCaseCreateAndGet_Classic` в `code/components/business/internal/usecase/user_usecase_test.go`.

Лондонский стиль:
- Максимум моков, проверяем взаимодействия и делегирование.
- Пример: `TestGameUseCaseProcessMoveDelegatesToMoveUseCase` в `code/components/business/internal/usecase/game_usecase_test.go`.

## Тесты на исключения (ожидаемые ошибки)
Идея: проверяем негативные сценарии, где результатом является ошибка.

Примеры:
- `TestUserUseCaseCreateRequiresIDGenerator` в `code/components/business/internal/usecase/user_usecase_test.go` — отсутствие генератора ID.
- `TestGameRepositorySave_RejectsInvalidID` в `code/components/data/internal/repository/localjson/game_repository_test.go`.

## Fixture (фикстуры)
Fixture — это подготовленный контекст для теста (репозиторий, контекст, тестовые данные).

Пример:
- `newUserRepoFixture` в `code/components/data/internal/repository/localjson/test_helpers_test.go`.
  Он создает storage и репозиторий пользователя для тестов.

## Data Builder (паттерн Builder)
Builder позволяет удобно собирать тестовые объекты с нужными полями.

Пример:
- `NewUserBuilder()` в `code/components/business/internal/usecase/test_helpers_test.go`.
- Использование в `TestUserUseCaseUpdateFields`:
  строим пользователя цепочкой `WithID/WithName/WithRating`.

## Object Mother (Fabric)
Object Mother — фабрика, которая создает типичные “готовые” объекты.

Пример:
- `MotherUser()` в `code/components/business/internal/usecase/test_helpers_test.go`.
- `MotherMove()` в `code/components/data/internal/repository/localjson/test_helpers_test.go`.
  Эти функции возвращают стандартные объекты для тестов.

## Что значит “один public‑метод — минимум 2 теста”
Для каждого публичного метода:
- минимум 1 позитивный сценарий
- минимум 1 негативный сценарий

Примеры:
- `GameRepository.Save`: `TestGameRepositorySave_OK` (позитив),
  `TestGameRepositorySave_RejectsInvalidID` (негатив).
- `GameUseCase.ProcessMove`: `TestGameUseCaseProcessMoveDelegatesToMoveUseCase` (позитив),
  `TestGameUseCaseProcessMoveRequiresGameID` (негатив).

## Где смотреть отчеты тестов
Скрипт запуска:
`code/product/run_tests.sh`

Он формирует:
- `code/product/test-report/allure-report/` — HTML отчет Allure
- `code/product/test-report/*-coverage.html` — HTML coverage
