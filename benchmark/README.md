# Benchmark (Lab 3)

## Что бенчмарим
Webapp API (Go) с PostgreSQL. Тестируем реальный бизнес-поток: создание пользователя, создание игры, ход, получение истории, удаление игры.

## Инструменты
- Нагрузка: k6
- Метрики ресурсов: Prometheus + cAdvisor
- Графики: Python + matplotlib

## Сценарии
- **Degradation**: плавный рост нагрузки до пика
- **Peak**: удержание максимальной нагрузки
- **Recovery**: спад нагрузки после перегруза

## Метрики
- Latency (http_req_duration): p50/p75/p90/p95/p99, гистограмма и график во времени
- Ошибки (http_req_failed)
- Ресурсы по сервисам (webapp, postgres): CPU, RAM, disk read/write

## Запуск
Из корня репозитория:

```bash
RUNS=100 ./benchmark/run_benchmarks.sh
```

Быстрый прогон:

```bash
RUNS=1 ./benchmark/run_benchmarks.sh
```

## Ограничение ресурсов (фиксированные условия)
Можно ограничить ресурсы контейнеров:

```bash
WEBAPP_CPUS=1.0 WEBAPP_MEM=1g POSTGRES_CPUS=1.0 POSTGRES_MEM=1g K6_CPUS=1.0 K6_MEM=512m \
  RUNS=100 ./benchmark/run_benchmarks.sh
```

## Ускорение сборки (кэш Go)
Для ускорения сборки используется Docker BuildKit и кеш слоёв/модулей.
Кеш хранится в `benchmark/.docker-cache` и не меняет требование \"отдельный образ на каждый прогон\".

## Артефакты
- `benchmark/results/run_*/k6.json` — сырые метрики k6
- `benchmark/results/run_*/summary.json` — сводка по прогонам
- `benchmark/results/run_*/latency_*.png` — графики латентности
- `benchmark/results/run_*/resources_*.png` — графики ресурсов
- `benchmark/results/summary/summary.json` — агрегированная сводка по 100 прогонам
- `benchmark/results/summary/p95_trend.png` — тренд p95 по прогонам
