FROM golang:1.26.1-bookworm AS deps

RUN dpkg --add-architecture arm64

RUN apt-get update && apt-get install -y \
    gcc-mingw-w64-x86-64 \
    g++-mingw-w64-x86-64 \
    gcc \
    g++ \
    gcc-aarch64-linux-gnu \
    g++-aarch64-linux-gnu \
    libgl1-mesa-dev \
    xorg-dev \
    libgl1-mesa-dev:arm64 \
    libx11-dev:arm64 \
    libxrandr-dev:arm64 \
    libxxf86vm-dev:arm64 \
    libxi-dev:arm64 \
    libxcursor-dev:arm64 \
    libxinerama-dev:arm64 \
    git \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN go install fyne.io/fyne/v2/cmd/fyne@latest

WORKDIR /clip
COPY go.mod go.sum ./
RUN go mod download

FROM deps AS source

WORKDIR /clip
COPY . .
RUN fyne bundle -o bundled.go assets/

FROM source AS build-windows

RUN CC=x86_64-w64-mingw32-gcc \
    CXX=x86_64-w64-mingw32-g++ \
    CGO_ENABLED=1 \
    GOOS=windows \
    GOARCH=amd64 \
    go build \
    -ldflags="-H windowsgui -s -w" \
    -o /out/clip-windows-amd64.exe \
    .


FROM source AS build-linux-amd64

RUN CC=gcc \
    CXX=g++ \
    CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -ldflags="-s -w" \
    -o /out/clip-linux-amd64 \
    .

FROM source AS build-linux-arm64

RUN CC=aarch64-linux-gnu-gcc \
    CXX=aarch64-linux-gnu-g++ \
    CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=arm64 \
    PKG_CONFIG_LIBDIR=/usr/lib/aarch64-linux-gnu/pkgconfig \
    CGO_CFLAGS="-I/usr/aarch64-linux-gnu/include" \
    CGO_LDFLAGS="-L/usr/aarch64-linux-gnu/lib" \
    go build \
    -ldflags="-s -w" \
    -o /out/clip-linux-arm64 \
    .

FROM scratch AS export
COPY --from=build-windows     /out/clip-windows-amd64.exe  /clip-windows-amd64.exe
COPY --from=build-linux-amd64 /out/clip-linux-amd64        /clip-linux-amd64
COPY --from=build-linux-arm64 /out/clip-linux-arm64        /clip-linux-arm64