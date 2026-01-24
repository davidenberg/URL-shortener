FROM golang:1.25.5

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build ./cmd/url-shortener

EXPOSE 8080

CMD ["./url-shortener"]