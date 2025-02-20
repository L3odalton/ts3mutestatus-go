FROM golang:1.24.0-alpine AS builder
WORKDIR /app
COPY . .
# Create the go.mod file inside the container
RUN cd /app && \
    go mod init ts3mutestatus-go && \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ts3mutestatus .

FROM scratch
COPY --from=builder /app/ts3mutestatus /
ENTRYPOINT ["/ts3mutestatus"]