FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o podman-swarm-agent ./cmd/agent

FROM alpine:latest

RUN apk --no-cache add ca-certificates podman

WORKDIR /root/

COPY --from=builder /build/podman-swarm-agent .

EXPOSE 8080 7946 80

CMD ["./podman-swarm-agent"]
