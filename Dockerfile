FROM golang:1.22.1

WORKDIR /app

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /backend ./cmd/run/run.go

EXPOSE 8181

CMD ["/backend"]
