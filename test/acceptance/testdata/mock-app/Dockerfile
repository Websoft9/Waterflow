FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY main.go .
RUN go build -o mock-app main.go

FROM alpine:3.19

RUN apk add --no-cache curl

WORKDIR /app
COPY --from=builder /app/mock-app .

ENV PORT=8080
ENV APP_VERSION=1.0.0

EXPOSE 8080

CMD ["./mock-app"]
