FROM golang:1.24-alpine

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . .

RUN go build -o main main.go

EXPOSE 8080

CMD [ "./main" ]