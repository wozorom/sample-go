FROM --platform=linux/amd64 docker-hub.artifactory.playticorp.com/golang:1.21-bookworm as builder

COPY ./deployments/certs /usr/local/share/ca-certificates/
RUN update-ca-certificates
COPY ./deployments/scripts /opt/scripts/
RUN chmod +x /opt/scripts/*.sh

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . ./

RUN GOOS=linux GOARCH=amd64 go build -v -o server ./cmd/main.go

FROM --platform=linux/amd64 docker-hub.artifactory.playticorp.com/debian:bookworm-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/go.mod /go.mod
COPY --from=builder /opt/scripts /opt/scripts/
COPY --from=builder /app/server /server

EXPOSE 8080 8081

CMD ["/server"]

