FROM golang:1.27-alpine

WORKDIR /app

COPY . .

RUN go mod tidy && CGO_ENABLED=0 go build -o /app/bin/server ./cmd/server

EXPOSE 8080

CMD ["sh", "-c", "/app/bin/server"]
