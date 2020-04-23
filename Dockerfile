FROM golang:alpine as builder

RUN apk update && apk add git

COPY . $GOPATH/src/github.com/AlbinoDrought/creamy-stuff
WORKDIR $GOPATH/src/github.com/AlbinoDrought/creamy-stuff

RUN go get -d -v

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /go/bin/creamy-stuff

FROM scratch

COPY --from=builder /go/bin/creamy-stuff /go/bin/creamy-stuff
ENTRYPOINT ["/go/bin/creamy-stuff"]
