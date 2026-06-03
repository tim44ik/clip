#!/bin/bash
set -e

# Запуск виртуального дисплея
Xvfb :99 -screen 0 1280x720x24 &
export DISPLAY=:99

# Ожидание запуска Xvfb
sleep 2

# Запуск VNC-сервера (без пароля, на все интерфейсы)
x11vnc -display :99 -forever -nopw -shared &

# Запуск noVNC веб-сервера (порт 6080)
/opt/noVNC/utils/novnc_proxy --listen 6080 --vnc localhost:5900 &

# Ожидание полного старта
sleep 3

# Запуск GUI-приложения
/app/clip-app &

# Бесконечное ожидание, чтобы контейнер не завершился
wait