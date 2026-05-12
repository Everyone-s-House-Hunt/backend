FROM golang:1.23-alpine

RUN apk update && apk add git

WORKDIR /app

RUN go install github.com/air-verse/air@latest

COPY . .

CMD ["air"]