# PPO Project Context (quick restore)

Этот файл — короткая “шпаргалка”, чтобы быстро восстановить контекст проекта в новом чате/сессии.

## Что это за проект
- Система визуализации ходов (шашки/шахматы) с AI‑партнёром и анимацией перемещений на доске.
- Основной поток: ввод хода → `GameUseCase` валидирует/сохраняет → `MoveUseCase` получает ход оппонента (OpenAI) → `AnimationUseCase` строит пошаговую анимацию.
- Правила игры (как в текущей реализации): обязательное взятие при наличии и обязательное продолжение цепочки взятий, пока это возможно для той же фигуры. Король (king) ходит и бьет только на 1/2 клетки по диагонали в любую сторону (без “дальних” ходов).
- Анимация для “физической доски”: при взятии сначала выводим побитую шашку по неиграбельным клеткам в сторону её игрока, затем двигаем атакующую шашку на клетку побитой и на клетку приземления; при серии взятий повторяем по каждой пешке.

## Архитектура (слои)
- **Domain**: `User`, `Game`, `Move`, `BoardState`, `Position`, `Animation`, инварианты в конструкторах (`NewMove`, `Game.Finish`).
- **Use cases**:
  - `UserUseCase`: `Create/Get/UpdateFields` (ID генерируется внутри, рейтинг/серия побед стартуют с 0).
  - `GameUseCase`: `New/ProcessMove/FinishGame/GenerateHistory/GetChessboard/UpdateBoardState` (ID/номера ходов генерируются).
  - `MoveUseCase`: `AddUserMove/GetOpponentMove` (статус `pending → in_progress`, есть fallback для “плохого” path от LLM).
  - `AnimationUseCase`: `AnimateMove` (обработка блокировок, “вывоз” жертвы).
- **Ports**:
  - Репозитории: `internal/port/repo` (`UserRepository`, `GameRepository`, `MoveRepository`).
  - `BoardStateReader`, `OpponentMoveProvider` (OpenAI/mock), `IDGenerator`.
- **Infrastructure**:
  - PostgreSQL: `internal/repository/postgres` (транзакции на `Save/UpdateBoardState`).
  - Local JSON storage: `internal/repository/localjson` (через `DATA_SOURCE`, без БД).
  - OpenAI: `internal/adapter/openai` (требует `OPENAI_MODEL` и `OPENAI_BASE_URL` при наличии ключа).
  - DI: `internal/app` + `internal/config`.
  - UUID: `internal/service/idgen`.

## Модули и “точки входа”
- Webapp: `PPO/code/apps/webapp/cmd/webapp` (HTTP сервер + статика webui).
- Плагины:
  - Data plugin: `PPO/code/components/data/cmd/plugin` (buildmode=plugin → `data.so`).
  - Business plugin: `PPO/code/components/business/cmd/plugin` (buildmode=plugin → `business.so`).
- Web UI (frontend build): `PPO/code/components/webui` (Vite → `dist/`).

## Конфиг и переменные окружения
Webapp config: `PPO/code/apps/webapp/internal/config/config.go`
- `POSTGRES_DSN` или `DATA_SOURCE` (можно указать json-конфиг).
- `DATA_PLUGIN_PATH`, `BUSINESS_PLUGIN_PATH`.
- `OPENAI_API_KEY`, `OPENAI_MODEL`, `OPENAI_BASE_URL`.
- `HTTP_TIMEOUT` (duration или секунды).
- `WEBAPP_ADDR` (адрес сервера, например `:8081`).

## Быстрые команды
Из корня `/root/opt/ppo`:
- Запуск (Postgres): `make -C PPO run`
- Запуск (локальное хранилище): `make -C PPO run-local`
- Поменять порт: `WEBAPP_ADDR=:18080 make -C PPO run` (или `run-local`)

## Тесты
- Все тесты:  
  `POSTGRES_DSN=postgres://... go test ./... -v -count=1`
- Только репозитории:  
  `go test ./internal/repository/postgres -v -count=1`
- Live OpenAI (опционально):  
  `RUN_OPENAI_LIVE_TEST=1 OPENAI_API_KEY=... OPENAI_MODEL=gpt-4o-mini go test ./internal/integration -run TestLiveOpenAIIntegration -v -count=1`

## БД
- Миграции/схема: `PPO/code/components/data/migrations/001_init.sql` (таблицы `users`, `games`, `game_board_states`, `moves`).
- Auth миграция: `PPO/code/components/data/migrations/002_add_user_credentials.sql` (таблица `user_credentials`).
- В тестах используется `SetupTestDB` (миграции + TRUNCATE между тестами).
