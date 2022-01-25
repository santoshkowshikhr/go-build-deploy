FROM golang:1.17-alpine

WORKDIR /go/src/app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /build-deploy

ENTRYPOINT [ "/build-deploy" ]