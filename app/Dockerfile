FROM golang:1.23

WORKDIR /app

RUN curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s -- -b /usr/local/bin

COPY . .

#RUN go mod download

EXPOSE 8080

CMD ["air"]