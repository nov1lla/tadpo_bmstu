# Лабораторная 2: интеграционные и E2E тесты

## Что сделано по требованиям
- Интеграционные тесты для компонента доступа к данным и бизнес‑логики.
- E2E‑тест демонстрационного сценария через HTTP API.
- CI/CD пайплайн с порядком: unit → integration → e2e.
- Docker‑контейнер для запуска тестов, отдельный контейнер с БД.
- Генерация отчета Allure и сохранение истории (тренды).
- Скрипт для имитации E2E сценария curl‑запросами с tcpdump‑захватом трафика.

## Где тесты
- Data integration (Postgres): `code/components/data/internal/repository/postgres/*_it_test.go`
- Business integration: `code/components/business/internal/integration/business_flow_it_test.go`
- E2E: `code/tests/e2e/e2e_test.go`

## Как запустить локально
- Unit тесты: `./code/product/run_tests.sh`
- Unit → Integration → E2E: `./code/product/run_ci_tests.sh`
- Захват трафика (curl + tcpdump): `./code/product/e2e_traffic_capture.sh`

## CI/CD
- Workflow: `.github/workflows/lab2.yml`
- Запуск в Docker через `docker/docker-compose.test.yml`
- Порядок стадий: unit → integration → e2e
- Если стадия упала, последующие помечаются как skipped в Allure
- История Allure сохраняется в `code/product/test-report/allure-history`

## Про требования по хранилищу
- Для интеграционных/E2E тестов создается отдельная база данных с уникальным именем.
- Миграции накатываются последовательно из `code/components/data/migrations`.
- БД удаляется в cleanup, что обеспечивает откат состояния после прогона.

## Про шины/сессии
- Message broker/очереди не используются.
- Сессии пользователей не хранятся, поэтому дополнительных действий по их завершению не требуется.
