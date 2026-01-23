# ЛР3: соответствие требованиям и демонстрация

Ниже — все пункты задания и требований ЛР3: где реализовано и как показать на защите.

## Задание

Составить набор сценариев (benchmark) для оценки производительности выбранного объекта.
- Объект: Webapp API + PostgreSQL
- Почему: реальные бизнес‑операции (создание пользователя → игра → ход → история → удаление)
- Док: `benchmark/README.md`, `LAB3_REPORT.md`

## Требования

1) **Не менее 100 испытаний и статистика**
- Где: `benchmark/run_benchmarks.sh` (переменная `RUNS=100` по умолчанию)
- Как показать:
  - `RUNS=100 ./benchmark/run_benchmarks.sh`
  - Итоговый агрегат: `benchmark/results/summary/summary.json`

2) **Несколько сценариев (degradation/peak/recovery)**
- Где: `benchmark/k6/scenarios.js`
- Как показать:
  - в файле есть `scenarios: degradation/peak/recovery`

3) **Отдельный docker‑образ на прогон, одна конфигурация**
- Где: `benchmark/run_benchmarks.sh`, `docker/benchmark/Dockerfile`
- Как показать:
  - каждый прогон строит image с уникальным тегом и `BENCH_RUN_ID`

4) **График во времени, перцентили, гистограмма (0.5/0.75/0.9/0.95/0.99)**
- Где: `benchmark/scripts/analyze_run.py`
- Как показать:
  - `benchmark/results/run_*/latency_timeseries.png`
  - `benchmark/results/run_*/latency_percentiles.png`
  - `benchmark/results/run_*/latency_histogram.png`

5) **2–3 параметра**
- Где: `benchmark/k6/scenarios.js` (endpoint tags)
- Как показать:
  - низкая нагрузка/серилизация: `users.create`
  - средняя: `games.move`
  - тяжелая: `games.create`, `games.moves`
  - per‑endpoint статистика в `benchmark/results/run_*/summary.json`

6) **Снятие ресурсов CPU/RAM/IO + графики + JSON**
- Где: `docker/benchmark/docker-compose.bench.yml` (cAdvisor + Prometheus),
  `benchmark/scripts/fetch_resources.py`, `benchmark/scripts/analyze_run.py`
- Как показать:
  - `benchmark/results/run_*/resources_*.png`
  - `benchmark/results/run_*/summary.json` содержит `resources` с min/max/mean

7) **Готовый инструмент нагрузки**
- Где: `benchmark/k6/scenarios.js`
- Как показать:
  - `k6` контейнер запускается в `benchmark/run_benchmarks.sh`

8) **Свой генератор — допустимо**
- Не используется, т.к. выбран k6

9) **Не использовать готовый бенчмарк «из коробки»**
- Реализован свой сценарий в `benchmark/k6/scenarios.js`

10) **Один хост, фиксированные ресурсы**
- Где: `benchmark/run_benchmarks.sh`, `docker/benchmark/docker-compose.bench.yml`
- Как показать:
  - `WEBAPP_CPUS/WEBAPP_MEM` и `POSTGRES_CPUS/POSTGRES_MEM` фиксируют ресурсы
  - `K6_CPUS/K6_MEM` для генератора нагрузки

11) **Ожидаемый способ использования (a–f)**
- Где: `benchmark/run_benchmarks.sh`
- Как показать:
  a. Поднять docker‑образ: `docker/benchmark/Dockerfile` (build)
  b. Запустить тесты: `k6 run ...`
  c. Собрать статистику: `k6.json`, `resources.json`, `summary.json`
  d. Повторить 100 раз: `RUNS=100`
  e. Итоговый отчет: `benchmark/results/summary/summary.json`, `p95_trend.png`
  f. Альтернативный объект: заменить Dockerfile/образ и повторить

## Быстрый показ

```bash
RUNS=1 ./benchmark/run_benchmarks.sh
ls -lh benchmark/results/run_*/summary.json
```

## Полный прогон

```bash
RUNS=100 ./benchmark/run_benchmarks.sh
```
