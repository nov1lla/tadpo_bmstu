# Лабораторная работа 3 — Benchmark

## Контекст проекта
Проект — система визуализации ходов в игре (шашки/шахматы) с бизнес-логикой (use case) и HTTP API. Основной поток: создание пользователя → создание игры → ход → история ходов. Этот поток и выбран как основной объект измерения.

## Что измеряем
**Объект:** Webapp API (Go) + PostgreSQL.

**Почему так:** сравнение с локальным хранилищем не использовано (не гарантирована корректность/стабильность локального режима). Мы измеряем производительность основной ветки приложения на стабильной инфраструктуре (PostgreSQL).

## Сценарии (несколько сценариев)
1. **Degradation** — постепенное наращивание нагрузки до пика.
2. **Peak** — удержание максимальной нагрузки на стабильном уровне.
3. **Recovery** — снижение нагрузки после перегруза.

## Метрики
- **Latency** (`http_req_duration`): p50/p75/p90/p95/p99 + график во времени + гистограмма.
- **Throughput** (`http_reqs_rate`) и **ошибки** (`http_req_failed_rate`).
- **Ресурсы по сервисам** (webapp, postgres): CPU, RAM, disk read/write.

## Инструменты
- Нагрузка: **k6** (популярный и простой инструмент)
- Ресурсы: **Prometheus + cAdvisor**
- Аналитика/графики: **Python + matplotlib**

## Как запустить
```bash
RUNS=100 ./benchmark/run_benchmarks.sh
```

Быстрый запуск:
```bash
RUNS=1 ./benchmark/run_benchmarks.sh
```

Фиксация ресурсов (одинаковые условия):
```bash
WEBAPP_CPUS=1.0 WEBAPP_MEM=1g POSTGRES_CPUS=1.0 POSTGRES_MEM=1g K6_CPUS=1.0 K6_MEM=512m \
  RUNS=100 ./benchmark/run_benchmarks.sh
```

## Артефакты
- `benchmark/results/run_*/k6.json` — сырые метрики k6
- `benchmark/results/run_*/k6_summary.json` — сводка k6
- `benchmark/results/run_*/summary.json` — агрегированные показатели прогона
- `benchmark/results/run_*/latency_*.png` — графики latency
- `benchmark/results/run_*/resources_*.png` — графики ресурсов
- `benchmark/results/summary/summary.json` — агрегат по 100 прогонам
- `benchmark/results/summary/p95_trend.png` — тренд p95 по прогонам

## Альтернативный объект
Если нужно сравнение, используется тот же сценарий и тот же пайплайн, но с другим образом webapp (например, отдельный Dockerfile или альтернативная конфигурация). Это гарантирует одинаковые условия и сопоставимость результатов.

## Соответствие требованиям
1. **100 испытаний**: `RUNS=100` в `benchmark/run_benchmarks.sh`.
2. **Несколько сценариев**: degradation/peak/recovery в `benchmark/k6/scenarios.js`.
3. **Отдельный docker-образ на прогон**: в `benchmark/run_benchmarks.sh` каждый прогон собирает новый image tag (`BENCH_RUN_ID`).
4. **Графики latency**: 
   - во времени — `latency_timeseries.png`
   - перцентили — `latency_percentiles.png`
   - гистограмма — `latency_histogram.png`
5. **2–3 параметра**: замеряются разные типы запросов с тегами endpoint:
   - небольшой JSON: `users.create`,
   - средний: `games.move`,
   - тяжелый: `games.create`/`games.moves`.
6. **Ресурсы CPU/RAM/IO**: Prometheus+cAdvisor и графики `resources_*.png` + JSON summary.
7. **Генератор нагрузки**: k6.
8. **Одинаковые условия**: ресурсы фиксируются через `WEBAPP_CPUS/WEBAPP_MEM/...`, каждый прогон — отдельная БД и compose-проект.
9. **Несколько запусков и параллельные разработчики**: уникальные project name и порты на каждый прогон.
