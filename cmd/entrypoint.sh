#!/bin/bash
set -e

Xvfb :99 -screen 0 1280x720x24 &
export DISPLAY=:99

sleep 2

x11vnc -display :99 -forever -nopw -shared &

/opt/noVNC/utils/novnc_proxy --listen 6080 --vnc localhost:5900 &

sleep 3

/app/clip-app &

wait