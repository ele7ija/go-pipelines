FROM golang:1.17.2-alpine

WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .
RUN go build -o /go-pipelines ./cmd/go-pipelines/main.go

ENV USER_REGO_PATH=/app/user/rego
ENV LOAD_REGO_PATH=/app/policy/rego

CMD [ "/go-pipelines" ]