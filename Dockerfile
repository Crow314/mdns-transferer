FROM golang:1.19

RUN apt-get update && apt-get install -y tcpdump

WORKDIR /go/src/app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

CMD sh -c 'go test ./pkg/conn && go test ./pkg/mdns'
