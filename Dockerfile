FROM golang:1.24.2

WORKDIR /wallet-backend

RUN go install github.com/air-verse/air@latest

COPY . .
RUN go mod tidy