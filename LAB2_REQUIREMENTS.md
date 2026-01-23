# ЛР2: соответствие требованиям и демонстрация

Ниже — все пункты задания и требований ЛР2: где реализовано и как показать на защите.

## Задание

1) Интеграционные тесты для data и business компонентов
- Где: `code/components/data/internal/repository/postgres/*_it_test.go`, `code/components/business/internal/integration/business_flow_it_test.go`
- Как показать:
  - `go test ./... -tags integration -count=1` в `code/components/data`
  - `go test ./... -tags integration -count=1` в `code/components/business`

2) E2E тест демонстрационного сценария
- Где: `code/tests/e2e/e2e_test.go`
- Как показать:
  - `go test ./... -tags e2e -count=1` в `code/tests/e2e`

3) Запуск ЛР1 + ЛР2 в CI/CD
- Где: `.github/workflows/lab2.yml` + `code/product/run_ci_tests.sh`
- Как показать:
  - В GitHub Actions запускается workflow `lab-02-tests`
  - Порядок внутри `run_ci_tests.sh`: unit → integration → e2e

4) Имитация E2E через запросы + захват трафика
- Где: `code/product/e2e_traffic_capture.sh`
- Как показать:
  - `./code/product/e2e_traffic_capture.sh`
  - Артефакт: `code/product/traffic/e2e_traffic.pcap`

## Требования

1) Запуск тестов в Docker-контейнере, репозиторий клонируется
- Где: `docker/tests/Dockerfile`, `docker/tests/entrypoint.sh`, `docker/docker-compose.test.yml`
- Как показать:
  - В `entrypoint.sh` есть clone по `REPO_URL`/`REPO_REF`
  - CI: `docker compose -f docker/docker-compose.test.yml up ...` (в workflow)

2) Интеграционные тесты используют хранилище данных
- Где: `code/components/data/internal/repository/postgres/*_it_test.go`, `code/components/business/internal/integration/business_flow_it_test.go`
- Как показать:
  - В тестах используется Postgres через `POSTGRES_DSN`

3) Для тестов инициализируется отдельный инстанс хранилища
- Где: `code/components/data/internal/repository/postgres/test_helper.go`, `code/components/data/testsupport/postgres.go`
- Как показать:
  - Создается БД с уникальным суффиксом `*_test_<random>`

4) Инстанс хранилища поднимается скриптами/образом и откатывается
- Где: `code/components/data/migrations` + `runMigrations` в `test_helper.go` / `testsupport/postgres.go`
- Как показать:
  - Миграции накатываются последовательно
  - В cleanup БД удаляется (откат)

5) Порядок запуска: unit → integration → e2e
- Где: `code/product/run_ci_tests.sh`
- Как показать:
  - В скрипте последовательный запуск, e2e только после integration

6) Если этап упал, последующие не запускаются, отчёт генерируется, skipped
- Где: `code/product/run_ci_tests.sh`
- Как показать:
  - `write_skipped` создает JUnit для skipped
  - Allure генерится в любом случае

7) Тренды между прогонами
- Где: `code/product/run_ci_tests.sh` (копирование `allure-history`), `.github/workflows/lab2.yml` (upload/download артефакта)
- Как показать:
  - В Actions есть артефакт `allure-history`
  - После второго прогона в Allure видна история

8) При падении integration/e2e откатить хранилище
- Где: `test_helper.go` / `testsupport/postgres.go`
- Как показать:
  - `Cleanup()` всегда удаляет тестовую БД

9) При наличии service bus — очистка очередей
- Статус: не используется в проекте

10) При наличии хранения сессий — завершение сессий
- Статус: не используется в проекте

11) GUI не нужен в E2E
- Где: `code/tests/e2e/e2e_test.go` (HTTP API)

12) Интеграционные тесты можно запускать много раз подряд с тем же результатом
- Где: тесты используют уникальную БД и миграции
- Как показать:
  - два прогона подряд дают одинаковый результат

13) Параллельный локальный запуск разными разработчиками
- Где: уникальный суффикс БД в `SetupTestDB` / `SetupPostgres`
- Как показать:
  - два одновременных прогона используют разные БД

14) Тесты должны проходить успешно
- Как показать:
  - `./code/product/run_ci_tests.sh`

15) Рекомендации по структуре тестов
- Где:
  - integration: `*_it_test.go` с тегом `//go:build integration`
  - e2e: `e2e_test.go` с тегом `//go:build e2e`
