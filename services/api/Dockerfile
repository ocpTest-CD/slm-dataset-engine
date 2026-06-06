FROM golang:1.24-alpine AS build

WORKDIR /src
COPY services/api/go.mod services/api/go.sum* ./services/api/
WORKDIR /src/services/api
RUN go mod download

COPY services/api/ ./
RUN go build -o /out/slm-api ./cmd/server

FROM alpine:3.21

WORKDIR /app
RUN adduser -D -H -u 1001 appuser
COPY --from=build /out/slm-api /app/slm-api
COPY migrations /app/migrations

ENV LISTEN_ADDR=:8080
ENV MIGRATIONS_DIR=/app/migrations
USER appuser
EXPOSE 8080

CMD ["/app/slm-api"]
