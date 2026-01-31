# ЛР5: Трассировка и мониторинг (OpenTelemetry)

Этот проект — Go-приложение (webapp + плагины data/business). Для ЛР5 добавлены:
- **трассировка (distributed tracing)** через OpenTelemetry (OTel) в webapp (HTTP вход),
- **мониторинг ресурсов** контейнеров через cAdvisor + Prometheus,
- **режимы логирования** (обычный vs расширенный) и способ сравнить их ресурсные затраты,
- **артефакты** (traces + метрики) сохраняются как при CI-прогонах тестов (docker), так и при benchmark (k6).

## Термины (коротко, по делу)

- **Trace** — «цепочка» работы запроса (например HTTP запрос /api/...).
- **Span** — кусок trace (например обработка HTTP запроса). У спана есть `trace_id`, `span_id`, длительность, атрибуты.
- **OTLP** — протокол OpenTelemetry для отправки traces/metrics/logs (у нас используется OTLP HTTP -> коллектор).
- **Collector** — агент/сервис, который принимает OTLP и экспортирует дальше (у нас экспортирует в файл).
- **Monitoring (Prometheus/cAdvisor)** — сбор CPU/RAM/IO метрик контейнеров (у нас это нужно для сравнения «с»/«без»).

## Задание (что сделано)

### 1) Интеграция трассировки и мониторинга

Где реализовано:
- Инициализация OTel в webapp: `code/apps/webapp/internal/observability/otel.go`
- Конфиг OTel и логирования из env: `code/apps/webapp/internal/config/config.go`
- Оборачивание HTTP handler в OTel middleware: `code/apps/webapp/cmd/webapp/main.go`
- Мониторинг контейнеров (benchmark): `docker/benchmark/docker-compose.bench.yml`, `docker/benchmark/prometheus.yml`
- Коллектор трасс (benchmark): `docker/benchmark/otel-collector.yml`, `docker/benchmark/docker-compose.bench.yml`
- Мониторинг контейнеров (CI docker tests): `docker/docker-compose.test.yml`, `docker/tests/prometheus.yml`
- Коллектор трасс (CI docker tests): `docker/tests/otel-collector.yml`, `docker/docker-compose.test.yml`
- Экспорт метрик во время CI docker tests: `docker/tests/entrypoint.sh` → `code/product/test-report/monitoring/resources.json`

Как продемонстрировать:
- **Трассы в benchmark**: после прогона открой `benchmark/results/.../run_.../otel_traces.json`
- **Трассы в CI docker tests**: после `make test-ci-docker` открой `code/product/test-report/otel_traces.json`
- **Мониторинг в benchmark**: `benchmark/results/.../run_.../resources.json` + графики `resources_*.png`
- **Мониторинг в CI docker tests**: `code/product/test-report/monitoring/resources.json`

### 2) Сравнить ресурсы при включенной трассировке и без неё

Идея: запускаем одинаковый benchmark дважды:
- **trace_off**: `OTEL_ENABLED=0`
- **trace_on**: `OTEL_ENABLED=1`

Инструменты сравнения:
- Сбор ресурсов по контейнерам (CPU/RAM/IO): `benchmark/scripts/fetch_resources.py`
- Аггрегация прогонов: `benchmark/scripts/aggregate_runs.py` (теперь считает `average_resources`)
- Отчёт сравнения: `benchmark/scripts/compare_summaries.py` → markdown

Команды:
- `make bench-lab5-trace-off RUNS=15`
- `make bench-lab5-trace-on RUNS=15`
- `make bench-lab5-compare-trace`

Артефакт сравнения:
- `benchmark/results/lab5_compare_trace.md`

Замечание по объёму трасс:
- Для уменьшения размера `otel_traces.json` можно снизить сэмплинг:
  - `OTEL_SAMPLE_RATIO=0.1 make bench-lab5-trace-on RUNS=15`

### 3) Сравнить ресурсы при логировании по умолчанию и расширенном

Сделано 2 режима:
- **default**: `LOG_LEVEL=info` (короткая строка на запрос)
- **debug**: `LOG_LEVEL=debug` (дополнительно логируются request/response body с лимитом `LOG_MAX_BODY_BYTES`)

Где реализовано:
- HTTP-логирование и режимы: `code/apps/webapp/internal/api/server.go`

Команды:
- `make bench-lab5-log-default RUNS=15`
- `make bench-lab5-log-debug RUNS=15`
- `make bench-lab5-compare-logging`

Артефакт сравнения:
- `benchmark/results/lab5_compare_logging.md`

## Требования (чек‑лист)

### Требование 1
> Данные мониторинга доступны при запуске тестов из CI\CD и benchmark-теста и сохраняются…

Где:
- CI docker tests:
  - Traces: `code/product/test-report/otel_traces.json`
  - Monitoring: `code/product/test-report/monitoring/resources.json`
  - Генерация: `docker/tests/entrypoint.sh`
- Benchmark:
  - Traces: `benchmark/results/.../run_.../otel_traces.json`
  - Monitoring: `benchmark/results/.../run_.../resources.json` (+ `resources_*.png`)

Как проверить:
- Локально: `make test-ci-docker` и убедиться, что файлы появились в `code/product/test-report/...`
- Benchmark: `RUNS=1 OTEL_ENABLED=1 ./benchmark/run_benchmarks.sh` и проверить наличие `otel_traces.json` и `resources.json`

### Требование 2
> Учитывается процессорное время и память для трассировки/мониторинга…

Как реализовано:
- В benchmark ресурсы собираются **по сервисам** (webapp/postgres/otel-collector/prometheus/cadvisor) из cAdvisor через Prometheus.
- Сравнение «trace_off vs trace_on» показывает **дельту CPU/RAM** (включая вклад `otel-collector`).

Где посмотреть:
- `benchmark/results/lab5_trace_off/summary/summary.json` → `average_resources`
- `benchmark/results/lab5_trace_on/summary/summary.json` → `average_resources`
- `benchmark/results/lab5_compare_trace.md` (дельты)

### Требование 3
> Учитываются ресурсы логирования…

Как реализовано:
- Сравнение `LOG_LEVEL=info` vs `LOG_LEVEL=debug` на одинаковом benchmark.
- Метрики `write_bytes` в `resources.json` помогают увидеть рост дисковых записей.

Где посмотреть:
- `benchmark/results/lab5_compare_logging.md`
- `benchmark/results/lab5_log_*/summary/summary.json` → `average_resources`

### Требование 4
> Формируется отчёт, где можно сравнить ресурсы…

Где:
- `benchmark/results/lab5_compare_trace.md`
- `benchmark/results/lab5_compare_logging.md`

