FROM alpine:3.23 AS build
ARG TARGETARCH
ARG TARGETVARIANT
RUN apk update && \
    apk add --no-cache go git bash ca-certificates && \
    go version && \
    rm -rf /var/cache/apk/*

ENV GOPROXY="https://proxy.golang.org"
ENV CGO_ENABLED=0

RUN mkdir -p /app/prebid-cache/
WORKDIR /app/prebid-cache/
COPY ./ ./
RUN go mod vendor
RUN go mod tidy
ARG TEST="true"
RUN if [ "$TEST" != "false" ]; then ./validate.sh ; fi
RUN go build -mod=vendor -ldflags "-X github.com/prebid/prebid-cache/version.Ver=`git describe --tags` -X github.com/prebid/prebid-cache/version.Rev=`git rev-parse HEAD`" .
FROM alpine:3.23 AS release
LABEL maintainer="hans.hjort@xandr.com" 
RUN apk add --no-cache ca-certificates
WORKDIR /usr/local/bin/
COPY --from=build /app/prebid-cache/prebid-cache .
RUN chmod a+xr prebid-cache
COPY --from=build /app/prebid-cache/config.yaml .
RUN chmod a+r config.yaml
RUN addgroup -g 2001 -S prebidgroup && adduser -u 1001 -S -G prebidgroup prebid
USER prebid
EXPOSE 2424
EXPOSE 2525
ENTRYPOINT ["/usr/local/bin/prebid-cache"]
