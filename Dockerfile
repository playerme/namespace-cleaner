FROM golang:1.9.2-alpine3.7

RUN apk update && apk upgrade && apk add --no-cache bash git openssh

WORKDIR /go/src/cleaner
COPY . /go/src/cleaner
RUN go get -v

CMD go run main.go