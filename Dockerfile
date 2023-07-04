FROM ubuntu:22.10

RUN apt-get update
RUN apt-get install -y wget git gcc

RUN wget -P /tmp https://go.dev/dl/go1.17.6.linux-amd64.tar.gz

RUN tar -C /usr/local -xzf /tmp/go1.17.6.linux-amd64.tar.gz
RUN rm /tmp/go1.17.6.linux-amd64.tar.gz

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

WORKDIR /go/src/app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /build-deploy

ENTRYPOINT [ "/build-deploy" ]