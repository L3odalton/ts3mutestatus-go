FROM --platform=$BUILDPLATFORM golang:1.24.0-alpine AS builder
ARG TARGETARCH
ARG BUILDPLATFORM
WORKDIR /app
COPY . .
RUN cd /app && \
    go mod init ts3mutestatus-go && \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -ldflags="-s -w" -o ts3mutestatus .

FROM scratch
LABEL maintainer="Linus Baumann <keen.key5715@linus-baumann.de>"
LABEL description="TeamSpeak 3 mute status synchronization with Home Assistant"
LABEL version="1.0"
COPY --from=builder /app/ts3mutestatus /
ENTRYPOINT ["/ts3mutestatus"]