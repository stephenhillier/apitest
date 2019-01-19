FROM golang:1.11
LABEL "name"="apitest"
LABEL "version"="0.0.1"

RUN mkdir -p /go/src/apitest/
COPY . /go/src/apitest

RUN go install -v /go/src/apitest/

ENTRYPOINT ["/go/src/apitest/entrypoint.sh"]
