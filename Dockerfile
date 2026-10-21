ARG GOVERSION=1.25.10

FROM golang:${GOVERSION}-bookworm AS build
WORKDIR /app/prebid-cache/
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY ./ ./
ARG TEST="true"
RUN if [ "$TEST" != "false" ]; then ./validate.sh ; fi
ARG VERSION=""
ARG REVISION=""
RUN go build -trimpath -ldflags "\
    -X github.com/prebid/prebid-cache/version.Ver=${VERSION} \
    -X github.com/prebid/prebid-cache/version.Rev=${REVISION}" .

FROM ubuntu:22.04 AS release
LABEL maintainer="hans.hjort@xandr.com"
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
RUN addgroup --system --gid 2001 prebidgroup && adduser --system --uid 1001 --ingroup prebidgroup prebid
WORKDIR /usr/local/bin/
COPY --from=build --chmod=555 /app/prebid-cache/prebid-cache .
COPY --from=build --chmod=444 /app/prebid-cache/config.yaml .
USER prebid
EXPOSE 2424 2525
ENTRYPOINT ["/usr/local/bin/prebid-cache"]
