FROM golang:1.24-alpine as builder
WORKDIR /app
COPY . .
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOGC=off GOMEMLIMIT=20MiB go build -ldflags="-w -s" -o app ./cmd/server/main.go
# RUN CGO_ENABLED=0 GOOS=linux GOGC=off GOMEMLIMIT=20MiB go build  -o app ./cmd/server/main.go

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/app .

EXPOSE 9999
CMD ["./app"] 