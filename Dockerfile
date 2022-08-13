FROM --platform=linux/amd64 golang:1.18 as build-env

WORKDIR /go/src/app
ADD . /go/src/app

RUN go get -d -v ./...

RUN GOOS=linux GOARCH=amd64 go build -o /go/bin/app cmd/collector/main.go

FROM --platform=linux/amd64 gcr.io/distroless/base
COPY --from=build-env /go/bin/app /

USER 200

ENTRYPOINT ["/app"]
