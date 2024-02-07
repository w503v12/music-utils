FROM golang:1.22.0 AS builder

RUN mkdir /app
ADD . /app
WORKDIR /app

RUN CGO_ENABLED=1 GOOS=linux go build -o app cmd/main.go

FROM ubuntu:22.04 AS production

RUN apt update && apt install -y bash curl

COPY --from=builder /app .

EXPOSE 28542

CMD ["./app"]