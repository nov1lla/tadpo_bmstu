#!/usr/bin/env bash
set -euo pipefail

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

DRY_RUN=false
if [[ "${1:-}" == "--dry-run" ]]; then
    DRY_RUN=true
    echo -e "${YELLOW}Режим просмотра (ничего не будет удалено)${NC}"
fi

# Функция для выполнения команд с учётом dry-run
run_cmd() {
    if $DRY_RUN; then
        echo "[DRY RUN] $*"
    else
        echo -e "${GREEN}-->${NC} $*"
        eval "$@"
    fi
}

echo -e "${YELLOW}=== Очистка мусора от бенчмарков ===${NC}"

# 1. Определяем корневую директорию проекта (где лежит benchmark)
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
BENCHMARK_DIR="${SCRIPT_DIR}/benchmark"
RESULTS_DIR="${BENCHMARK_DIR}/results"
CACHE_DIR="${BENCHMARK_DIR}/.docker-cache"

# 2. Проверка наличия Docker и прав
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Ошибка: docker не найден${NC}" >&2
    exit 1
fi

if ! docker info &> /dev/null; then
    echo -e "${RED}Ошибка: нет доступа к Docker. Возможно, нужен sudo или добавление в группу docker.${NC}" >&2
    exit 1
fi

# 3. Показать текущее использование диска Docker
echo -e "\n${YELLOW}Текущее использование диска Docker:${NC}"
docker system df

# 4. Показать размер папок с результатами
if [ -d "$RESULTS_DIR" ]; then
    echo -e "\n${YELLOW}Размер папок с результатами:${NC}"
    du -sh "$RESULTS_DIR"/* 2>/dev/null || echo "Нет папок run_*"
fi

if [ -d "$CACHE_DIR" ]; then
    echo -e "\n${YELLOW}Размер кэша сборки:${NC}"
    du -sh "$CACHE_DIR"
fi

# 5. Запрос подтверждения
echo ""
read -p "Продолжить очистку? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Очистка отменена."
    exit 0
fi

# 6. Удаление папок run_* из benchmark/results (с подтверждением)
if [ -d "$RESULTS_DIR" ]; then
    echo -e "\n${YELLOW}Удаление папок run_* в ${RESULTS_DIR}...${NC}"
    # Найдём все папки, начинающиеся с run_
    RUN_DIRS=$(find "$RESULTS_DIR" -maxdepth 1 -type d -name 'run_*' 2>/dev/null || true)
    if [ -n "$RUN_DIRS" ]; then
        echo "Будут удалены:"
        echo "$RUN_DIRS"
        read -p "Удалить эти папки? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            run_cmd "rm -rf $RUN_DIRS"
        else
            echo "Пропуск удаления папок."
        fi
    else
        echo "Нет папок run_* для удаления."
    fi
fi

# 7. Удаление локального кэша .docker-cache
if [ -d "$CACHE_DIR" ]; then
    echo -e "\n${YELLOW}Удаление локального кэша сборки ${CACHE_DIR}...${NC}"
    read -p "Удалить эту папку? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        run_cmd "rm -rf $CACHE_DIR"
    else
        echo "Пропуск удаления кэша."
    fi
fi

# 8. Очистка кэша BuildKit (docker builder prune)
echo -e "\n${YELLOW}Очистка кэша сборки BuildKit...${NC}"
run_cmd "docker builder prune -f"

# 9. Удаление неиспользуемых образов (все, не только dangling)
echo -e "\n${YELLOW}Удаление всех неиспользуемых образов...${NC}"
run_cmd "docker image prune -a -f"

# 10. Удаление остановленных контейнеров
echo -e "\n${YELLOW}Удаление остановленных контейнеров...${NC}"
run_cmd "docker container prune -f"

# 11. Удаление неиспользуемых томов
echo -e "\n${YELLOW}Удаление неиспользуемых томов...${NC}"
run_cmd "docker volume prune -f"

# 12. Очистка логов Docker (осторожно!)
echo -e "\n${YELLOW}Очистка логов контейнеров (опционально, может затронуть работающие контейнеры)${NC}"
read -p "Удалить все файлы логов Docker (/var/lib/docker/containers/*/*.log)? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    # Проверяем, что у нас есть права на запись в /var/lib/docker/containers
    if [ -w /var/lib/docker/containers ]; then
        run_cmd "sudo find /var/lib/docker/containers -name '*.log' -delete"
    else
        echo -e "${RED}Нет прав на запись в /var/lib/docker/containers. Попробуйте с sudo.${NC}"
    fi
else
    echo "Пропуск очистки логов."
fi

# 13. Итоговый отчёт
echo -e "\n${GREEN}Очистка завершена!${NC}"
echo -e "${YELLOW}Использование диска Docker после очистки:${NC}"
docker system df

if [ -d "$RESULTS_DIR" ]; then
    echo -e "\n${YELLOW}Оставшиеся папки в ${RESULTS_DIR}:${NC}"
    ls -ld "$RESULTS_DIR"/* 2>/dev/null || echo "Пусто"
fi\
