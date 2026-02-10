# Stage 1: Build Go binary
FROM golang:1.23-alpine AS go-builder

RUN apk add --no-cache git
WORKDIR /build
COPY go.mod ./
RUN go mod download || true
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o wubot ./cmd/wubot

# Stage 2: Runtime
FROM eclipse-temurin:22-jre-alpine

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy Go binary
COPY --from=go-builder /build/wubot .

# Copy pre-built JAR (place wupacket.jar in project root before building)
COPY wupacket.jar ./wupacket.jar

# Default config path
VOLUME ["/app/config.yaml"]

ENTRYPOINT ["./wubot"]
CMD ["--headless", "--config", "/app/config.yaml"]
