FROM golang:1.16.0-alpine

# Build the app, dependencies first
COPY go.mod go.sum /app/
WORKDIR /app
RUN go mod download

COPY . /app
ENV CGO_ENABLED=0
RUN go build -o main
RUN go test ./...

# ---
FROM alpine:3.13.1 AS dist

# Dependencies
RUN apk add --no-cache ca-certificates

# Add pre-built application
COPY --from=0 /app/main /app

EXPOSE 8080
ENTRYPOINT [ "/app" ]
LABEL org.opencontainers.image.source=https://github.com/taiidani/discord-manners
