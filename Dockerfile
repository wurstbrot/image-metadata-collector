FROM --platform=linux/amd64 golang:1.19 as build-env

WORKDIR /go/src/app
ADD . /go/src/app

RUN go get -d -v ./...

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /go/bin/app cmd/collector/main.go && \
    GOARCH=amd64 GOOS=linux go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.3.0 && \
    GOARCH=amd64 GOOS=linux cyclonedx-gomod mod -json=true -output /bom.json


FROM --platform=linux/amd64 gcr.io/distroless/static-debian11
COPY --from=build-env /go/bin/app /
COPY --from=build-env /bom.json /bom.json

USER 1001

CMD ["/app"]
