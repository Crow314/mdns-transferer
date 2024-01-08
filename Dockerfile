FROM golang:1.19

RUN apt-get update && apt-get install -y tcpdump

WORKDIR /go/src/app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

CMD sh -c 'go test ./cmd/proxy && go test ./pkg/conn'
