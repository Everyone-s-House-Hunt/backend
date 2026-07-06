FROM golang:1.25

WORKDIR /app

ENV TZ=Asia/Tokyo

RUN curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s -- -b /usr/local/bin

COPY . .

RUN go mod tidy

EXPOSE 8080

CMD ["air"]