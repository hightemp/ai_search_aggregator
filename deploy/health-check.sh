#!/bin/bash

echo "🔍 Проверка состояния сервисов..."

# Функция для проверки HTTP ответа
check_service() {
    local name=$1
    local url=$2
    local expected_code=${3:-200}
    
    echo -n "Проверяем $name ($url)... "
    
    response=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 --max-time 10 "$url" 2>/dev/null)
    
    if [ "$response" = "$expected_code" ]; then
        echo "✅ OK ($response)"
        return 0
    else
        echo "❌ FAIL ($response)"
        return 1
    fi
}

# Проверяем сервисы
echo ""
check_service "Frontend" "http://localhost:8082" 200
check_service "Backend Health" "http://localhost:9080/healthz" 200
check_service "SearxNG" "http://localhost:8081" 200
check_service "Nginx Proxy" "http://localhost:9081" 200

echo ""
echo "🔗 Проверка API через прокси..."
check_service "API через nginx" "http://localhost:9081/healthz" 200

echo ""
echo "📊 Проверка логов backend..."
echo "Последние 10 строк логов backend:"
docker compose -f /home/hightemp/Projects/go_links_seacher/deploy/docker-compose.yml logs --tail=10 backend

echo ""
echo "📊 Проверка логов nginx..."
echo "Последние 5 строк логов nginx:"
docker compose -f /home/hightemp/Projects/go_links_seacher/deploy/docker-compose.yml logs --tail=5 proxy