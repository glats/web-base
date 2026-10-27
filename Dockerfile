FROM golang:1.25.8-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/web ./cmd/web

FROM debian:bookworm-slim

WORKDIR /app
RUN mkdir -p /data && chown 10001:10001 /data
COPY --from=build /out/web /app/web
USER 10001:10001
ENV HTTP_ADDR=:8080 DATABASE_PATH=/data/app.db LOG_PRETTY=false
EXPOSE 8080
ENTRYPOINT ["/app/web"]
