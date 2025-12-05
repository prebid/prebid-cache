FROM alpine:3.23 AS build

RUN apk update && \
    apk add --no-cache go=1.25.5-r0 git bash ca-certificates && \
    rm -rf /var/cache/apk/*

ENV GOPROXY="https://proxy.golang.org" \
    CGO_ENABLED=0

WORKDIR /app/prebid-cache/

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go mod vendor

ARG TEST="true"
RUN if [ "$TEST" != "false" ]; then ./validate.sh ; fi

RUN go build -mod=vendor -ldflags "-X github.com/prebid/prebid-cache/version.Ver=$(git describe --tags 2>/dev/null || echo 'dev') -X github.com/prebid/prebid-cache/version.Rev=$(git rev-parse HEAD 2>/dev/null || echo 'unknown')" .
FROM alpine:3.23 AS release
LABEL maintainer="hans.hjort@xandr.com"

RUN apk add --no-cache ca-certificates && \
    rm -rf /var/cache/apk/*

RUN addgroup -g 2001 -S prebidgroup && \
    adduser -u 1001 -S -G prebidgroup prebid

WORKDIR /usr/local/bin/

COPY --from=build /app/prebid-cache/prebid-cache \
                  /app/prebid-cache/config.yaml ./

USER prebid
EXPOSE 2424 2525
ENTRYPOINT ["/usr/local/bin/prebid-cache"]
