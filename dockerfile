FROM golang:1.17 AS build

WORKDIR /app

COPY . .

RUN go mod download
<<<<<<< HEAD
RUN go build -o main .
=======
RUN GOOS=linux go build -o main .
>>>>>>> 0f0a6acf0769fd7b63f5e985d680007882cdd7e2
RUN chmod +x main

CMD ["./main"]
