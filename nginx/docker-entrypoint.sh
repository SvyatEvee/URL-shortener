#!/bin/sh
echo "Проверяем конфигурацию nginx..."
nginx -t
echo "Запускаем nginx..."
exec nginx -g 'daemon off;'