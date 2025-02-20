FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
# Create the go.mod file inside the container
RUN cd /app && \
    go mod init ts3mutestatus-go && \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ts3mutestatus .

FROM scratch
LABEL maintainer="Linus Baumann <keen.key5715@linus-baumann.de>"
LABEL description="TeamSpeak 3 mute status synchronization with Home Assistant"
LABEL version="1.0"
COPY --from=builder /app/ts3mutestatus /
ENTRYPOINT ["/ts3mutestatus"]