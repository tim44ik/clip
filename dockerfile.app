FROM golang:1.26.1-bookworm AS builder

RUN apt-get update && apt-get install -y \
    libgl1-mesa-dev \
    libx11-dev \
    libxrandr-dev \
    libxxf86vm-dev \
    libxi-dev \
    libxcursor-dev \
    libxinerama-dev \
    xorg-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /clip
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go install fyne.io/fyne/v2/cmd/fyne@latest && \
    fyne bundle -o bundled.go assets/ || true

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /clip-app ./cmd

FROM kalilinux/kali-rolling

RUN apt-get update && apt-get install -y locales && \
    echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && \
    locale-gen

ENV LANG=en_US.UTF-8 \
    LANGUAGE=en_US:en \
    LC_ALL=en_US.UTF-8 \
    DISPLAY=:99 \
    POSTGRES_HOST=db \
    POSTGRES_DB=cve_db \
    POSTGRES_USER=postgres \
    POSTGRES_PASSWORD=postgres \
    POSTGRES_PORT=5432

RUN apt-get update && apt-get install -y \
    kali-linux-headless \
    libgl1-mesa-dev \
    libx11-dev \
    libxrandr-dev \
    libxxf86vm-dev \
    libxi-dev \
    libxcursor-dev \
    libxinerama-dev \
    xorg \
    x11vnc \
    xvfb \
    fluxbox \
    wget \
    postgresql-client \
    && wget https://github.com/novnc/noVNC/archive/refs/tags/v1.4.0.tar.gz && \
    tar xzf v1.4.0.tar.gz && mv noVNC-1.4.0 /opt/noVNC && \
    rm v1.4.0.tar.gz && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

RUN echo '#!/bin/bash\n\
set -e\n\
\n\
Xvfb :99 -screen 0 1280x720x24 &\n\
export DISPLAY=:99\n\
\n\
sleep 2\n\
\n\
x11vnc -display :99 -forever -nopw -shared -xkb -rfbport 5900 &\n\
\n\
/opt/noVNC/utils/novnc_proxy --listen 6080 --vnc localhost:5900 &\n\
\n\
sleep 3\n\
\n\
/app/clip-app &\n\
\n\
wait' > /entrypoint.sh && chmod +x /entrypoint.sh

WORKDIR /app
COPY --from=builder /clip-app /app/clip-app

RUN useradd -m -s /bin/bash fyne && chown -R fyne:fyne /app

RUN mkdir /shared && chown fyne:fyne /shared
VOLUME /shared

EXPOSE 6080

USER fyne
ENTRYPOINT ["/entrypoint.sh"]