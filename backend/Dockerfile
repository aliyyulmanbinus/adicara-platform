FROM golang:1.27-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/adicara-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/adicara-migrate ./cmd/migrate

FROM alpine:3.23
RUN addgroup -S adicara && adduser -S -G adicara adicara
COPY --from=build /out/adicara-api /adicara-api
COPY --from=build /out/adicara-migrate /adicara-migrate
USER adicara
EXPOSE 8080
ENTRYPOINT ["/adicara-api"]

