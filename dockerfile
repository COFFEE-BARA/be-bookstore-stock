FROM golang:1.17 AS build

WORKDIR /app

COPY . .

RUN go mod download
RUN GOOS=linux go build -o main .
RUN chmod +x main

CMD ["./main"]
