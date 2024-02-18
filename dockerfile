FROM golang:1.17 AS build
WORKDIR /app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
COPY --from=build /app/main /usr/local/bin/main
CMD ["main"]