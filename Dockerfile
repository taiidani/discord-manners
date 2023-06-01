FROM alpine:3.18

# Dependencies
RUN apk add --no-cache ca-certificates

# Add pre-built application
COPY discord-manners /app

EXPOSE 8080
ENTRYPOINT [ "/app" ]
LABEL org.opencontainers.image.source=https://github.com/taiidani/discord-manners
