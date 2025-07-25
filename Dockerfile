FROM golang:1.21-alpine as builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o app ./cmd/server/main.go

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/app .
EXPOSE 9999
CMD ["./app"] 