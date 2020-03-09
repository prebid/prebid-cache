FROM 993930779011.dkr.ecr.us-east-1.amazonaws.com/sortable/loadconfig:0.1.3 as loadconfig
FROM golang as builder
RUN mkdir -p /go/src/github.com/prebid/prebid-cache
WORKDIR /go/src/github.com/prebid/prebid-cache
COPY . .

RUN go get -u github.com/golang/dep/cmd/dep
RUN dep ensure
RUN go test -v
RUN go build

FROM ubuntu:16.04
ARG DEBIAN_FRONTEND=noninteractive
RUN apt update
RUN apt install --assume-yes apt-utils
RUN apt install -y ca-certificates

RUN mkdir /app
COPY --from=builder /go/src/github.com/prebid/prebid-cache/prebid-cache /app/prebid-cache
COPY --from=loadconfig loadconfig /loadconfig
ADD ./config.yaml /app/

WORKDIR /app

EXPOSE 2525/tcp
EXPOSE 2424/tcp

CMD ["./prebid-cache"]
