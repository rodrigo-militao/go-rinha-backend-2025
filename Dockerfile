FROM golang:1.24-alpine as builder
WORKDIR /app
COPY . .
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOGC=off go build -ldflags="-w -s" -o main ./cmd/server/main.go
# RUN CGO_ENABLED=0 GOOS=linux GOGC=off GOMEMLIMIT=20MiB go build  -o main ./cmd/server/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/main .

RUN chmod +x /app/main

EXPOSE 9999
CMD ["./main"]