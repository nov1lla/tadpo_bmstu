# ЛР1: Unit-тесты и отчет (инструкции и теория с примерами из проекта)

Этот файл восстанавливает инструкции по ЛР1 и поясняет термины на примерах наших тестов.

## О чем проект (контекст для тестов)
Проект — система визуализации ходов (шашки/шахматы) с бизнес-логикой use-case и хранением данных. Основной поток: создание пользователя → создание игры → ход → история. Юнит-тесты покрывают слой доступа к данным и слой бизнес-логики.

## Как запустить тесты
Проект разбит на несколько Go-модулей, поэтому запускать тесты нужно в каждом модуле отдельно.

Юнит-тесты бизнес-логики:
```bash
(cd code/components/business && go test ./... -count=1)
```

Юнит-тесты слоя данных (локальное JSON-хранилище + Postgres-репозитории):
```bash
(cd code/components/data && go test ./... -count=1)
```

Юнит-тесты домена (правила шашек и проверка пути):
```bash
(cd code/sdk && go test ./... -count=1)
```

Если нужен запуск всех модулей подряд:
```bash
(cd code/components/business && go test ./... -count=1) \
  && (cd code/components/data && go test ./... -count=1) \
  && (cd code/sdk && go test ./... -count=1)
```

### Через Makefile
```bash
make test-unit
make test-unit-shuffle
make test-unit-offline
make test-unit-serial
```

### Postgres
Тесты в `code/components/data/internal/repository/postgres` используют DSN из `POSTGRES_DSN`.
Если БД недоступна, они корректно отмечаются как skipped. Для полного прогона:
```bash
export POSTGRES_DSN='postgres://admin:admin123@localhost:5432/ppo_cource?sslmode=disable'
(cd code/components/data && go test ./... -count=1)
```

## Как получить и просмотреть Allure-отчет
Готовый отчет уже лежит в `code/product/test-report/allure-report`.

Просмотр локально через статический сервер:
```bash
cd code/product/test-report/allure-report
python3 -m http.server 8080
```
Открой в браузере `http://localhost:8080`.

Если установлен `allure` CLI, можно открыть директорию напрямую:
```bash
allure open code/product/test-report/allure-report
```

### Генерация нового отчета через Makefile
```bash
make allure-generate
```
Команда запускает тесты, собирает JUnit-файлы в `code/product/test-report/allure-results` и генерирует Allure-отчет.
Потребуется установленный `allure` CLI.
JUnit генерируется через `gotestsum` (запускается как `go run gotest.tools/gotestsum@latest`).

### Просмотр отчета через Makefile
```bash
make allure-open
```
Открой в браузере `http://localhost:8080`.

## Запуск в случайном порядке (требование 10)
Go умеет перемешивать порядок тестов внутри пакета:
```bash
(cd code/components/business && go test ./... -count=1 -shuffle=on)
```
Для детерминированного перемешивания можно использовать seed:
```bash
(cd code/components/business && go test ./... -count=1 -shuffle=123)
```

## Запуск без доступа к интернету (требование 11)
Юнит-тесты используют моки/стабы и локальные httptest-серверы, поэтому они проходят без сети.
Чтобы явно запретить сетевые скачивания модулей:
```bash
GONOSUMDB='*' GOPROXY=off (cd code/components/business && go test ./... -count=1)
```
Важно: зависимости должны быть уже скачаны в модульном кеше.

## Сколько процессов запускается (требование 12)
Go запускает отдельный процесс на каждый пакет.

- `go test ./...` запускает несколько процессов (по одному на пакет).
- Количество пакетов, которые тестируются параллельно, задается флагом `-p`.
- Внутри одного пакета тесты выполняются в одном процессе и одном наборе goroutine.
- Параллельность тестов внутри пакета задается `t.Parallel()` и флагом `-parallel`.

Пример с ограничением параллельных пакетов:
```bash
(cd code/components/business && go test ./... -count=1 -p=1)
```

## Теория с примерами из нашего кода

### Arrange-Act-Assert (AAA)
Шаблон теста: подготовка → действие → проверка.
Пример (Arrange — подготовка репозиториев, Act — вызов `New`, Assert — проверка ID):
- `code/components/business/internal/usecase/game_usecase_test.go`
  - `TestGameUseCaseNewInitialisesBoard`

### Classic vs London (классический и лондонский стиль)
- Классический: реальные объекты без моков.
  - `code/components/data/internal/repository/localjson/localjson_repository_test.go`
  - `code/sdk/domain/checkers_rules_test.go`
- Лондонский (через моки/стабы):
  - `code/components/business/internal/usecase/game_usecase_test.go`
  - `code/components/business/internal/usecase/move_usecase_test.go`
  - `code/components/business/internal/usecase/user_usecase_test.go`

### Тесты на исключения/ошибки (требование 2)
Примеры ожидаемых ошибок:
- `code/components/business/internal/usecase/game_usecase_test.go`: `TestGameUseCaseProcessMoveRequiresGameID`
- `code/components/business/internal/usecase/move_usecase_test.go`: `TestMoveUseCaseGetOpponentMovePropagatesErrors`
- `code/components/business/internal/usecase/animate_move_test.go`: `TestAnimatorRejectsPathLeavingBoard`
- `code/components/business/internal/adapter/openai/client_test.go`: `TestClientSuggestMoveAPIError`

### Fixture (фикстура)
Фикстура — повторяемая подготовка окружения/данных.
- `code/components/data/internal/repository/localjson/localjson_repository_test.go`:
  `newTestStorage` создает временное хранилище.
- `code/components/data/internal/repository/postgres/test_helper.go`:
  `SetupTestDB` поднимает БД, гонит миграции, очищает таблицы.
- `code/components/business/internal/usecase/animate_move_test.go`:
  `newBoardStub` создает доску с начальными фигурами.

### Data Builder (строитель тестовых данных)
Data Builder — фабрика, которая принимает параметры и собирает объект.
Пример: `newBoardStub(size, placements)` строит доску с нужными позициями.
- `code/components/business/internal/usecase/animate_move_test.go`.

### Object Mother (фабрика готовых объектов)
Object Mother — объект/функция, создающая типовые сущности целиком.
Примеры — готовые объекты в тестах через хелперы:
- `newTestUser(...)` в `code/components/data/internal/repository/localjson/localjson_repository_test.go`
- `newTestGame(...)` и `newTestGameWithStart(...)` в `code/components/data/internal/repository/localjson/localjson_repository_test.go`
- `newTestMove(...)` в `code/components/data/internal/repository/localjson/localjson_repository_test.go`

Если понадобится, эти Object Mother можно вынести в отдельный `testsupport` пакет,
но сейчас это компактные хелперы внутри тестового файла.

## Покрытие методов (требование 1)
Ключевые публичные методы use-case и репозиториев покрыты позитивными и негативными сценариями.
Примеры:
- `GameUseCase.ProcessMove`: `TestGameUseCaseProcessMoveDelegatesToMoveUseCase` (позитив),
  `TestGameUseCaseProcessMoveRequiresGameID` (негатив).
- `MoveUseCase.GetOpponentMove`: `TestMoveUseCaseGetOpponentMove` (позитив),
  `TestMoveUseCaseGetOpponentMovePropagatesErrors` (негатив).
- `UserRepository.Save/Get/Update`: `TestUserRepository_SaveGetUpdate` (позитив),
  дубликат пользователя внутри того же теста (негатив).

## Требование 8 (CLI запуск)
Все тесты запускаются командами из раздела "Как запустить тесты".

## Требование 12 (процессы)
См. раздел "Сколько процессов запускается".

## Требование 13 (тесты должны проходить)
Текущие тесты проходят при наличии локального Postgres (для postgres-репозитория)
или будут skipped, если БД недоступна.

## Требование 14 (устойчивость и поддерживаемость)
В тестах:
- Используются явные моки портов (`repo.*Mock`, `port.OpponentMoveProviderMock`).
- Нет обращения к внешним системам в unit-тестах (OpenAI тестируется через `httptest`).
- Общая логика тестов читабельна и повторно использует фикстуры.

## Примечание по требованию 5
Если обнаружится метод, который невозможно вызвать одним действием в секции Act,
нужно рефакторить соответствующий класс. Пока таких случаев не найдено.
