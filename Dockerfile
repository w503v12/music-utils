FROM golang:1.19 AS builder

RUN mkdir /app
ADD . /app
WORKDIR /app

RUN CGO_ENABLED=1 GOOS=linux go build -o music-utils cmd/main.go

FROM alpine:latest AS production

RUN apk --no-cache add bash curl

COPY --from=builder /app .

EXPOSE 28542

CMD ["./music-utils"]