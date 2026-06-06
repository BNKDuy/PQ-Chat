FROM golang:1.26-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
    -ldflags="-s -w" \
    -o ./server ./cmd/server

FROM alpine:3.21

WORKDIR /app

COPY --from=build /app/server ./server

EXPOSE 8080

ENTRYPOINT ["./server"]