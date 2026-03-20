FROM golang:1.26

RUN apt-get update && apt-get install -y \
    pkg-config \
    libgl1-mesa-dev \
    libx11-dev \
    libxcursor-dev \
    libxrandr-dev \
    libxinerama-dev \
    libxi-dev \
    libxxf86vm-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY . .

RUN go build -o clip