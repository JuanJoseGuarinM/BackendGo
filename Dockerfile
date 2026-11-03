# syntax=docker/dockerfile:1
#
# Dockerfile MULTISTAGE: dos etapas en un solo archivo.
#   1. "build": imagen grande con Go para COMPILAR (~300 MB).
#   2. final:   imagen mínima que solo tiene el binario (~36 MB).
# Las herramientas de compilación no viajan a producción: imagen más pequeña,
# arranque más rápido y menos superficie de ataque.
#
# Un solo Dockerfile para los dos servicios, eligiendo con TARGET:
#   docker build -t orders-api:dev .
#   docker build -t catalog:dev --build-arg TARGET=catalog .

ARG GO_VERSION=1.27

# ---------- Etapa 1: compilar ----------
FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

# Primero SOLO go.mod y go.sum, y descargar dependencias. Docker guarda cada
# paso en caché: si cambias el código pero no las dependencias, este paso no
# se repite y el build es mucho más rápido.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGET=api
# CGO_ENABLED=0: binario estático, sin depender de librerías de C del sistema.
# -trimpath y -ldflags="-s -w": quitan rutas locales e info de depuración (más pequeño).
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/${TARGET}

# ---------- Etapa 2: imagen final ----------
# distroless/static: sin shell, sin gestor de paquetes, solo lo mínimo para
# correr un binario estático. ":nonroot" = corre con un usuario sin privilegios.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/app /app

# Buena práctica de seguridad: nunca correr como root dentro del contenedor.
# 65532 es el usuario "nonroot" de distroless. Va en NÚMERO porque Kubernetes,
# con runAsNonRoot: true, solo puede verificar un UID numérico.
USER 65532:65532

# Documenta el puerto; el valor real lo da la variable PORT.
ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/app"]
