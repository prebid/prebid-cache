FROM ubuntu:22.04 AS build
RUN apt-get update && \
    apt-get -y upgrade && \
    apt-get install -y --no-install-recommends wget ca-certificates
ENV GO_INSTALLER=go1.19.5.linux-amd64.tar.gz
WORKDIR /tmp
RUN wget https://dl.google.com/go/go1.23.1.linux-amd64.tar.gz && \
    tar -xf go1.23.1.linux-amd64.tar.gz && \
    mv go /usr/local
RUN mkdir -p /app/prebid-cache/
WORKDIR /app/prebid-cache/
ENV GOROOT=/usr/local/go
ENV PATH=$GOROOT/bin:$PATH
ENV GOPROXY="https://proxy.golang.org"
RUN apt-get update && \
    apt-get install -y --no-install-recommends git && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
ENV CGO_ENABLED=0
COPY ./ ./
RUN go mod tidy
RUN go mod vendor
ARG TEST="true"
# RUN if [ "$TEST" != "false" ]; then ./validate.sh ; fi
RUN go build -mod=vendor -ldflags "-X github.com/prebid/prebid-cache/version.Ver=`git describe --tags | sed 's/^v//'` -X github.com/prebid/prebid-cache/version.Rev=`git rev-parse HEAD`" .

FROM ubuntu:22.04 AS release
LABEL maintainer="hans.hjort@xandr.com" 
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
WORKDIR /usr/local/bin/
COPY --from=build /app/prebid-cache/prebid-cache .
RUN chmod a+xr prebid-cache
COPY --from=build /app/prebid-cache/config.yaml .
RUN chmod a+r config.yaml
RUN addgroup --system --gid 2001 prebidgroup && adduser --system --uid 1001 --ingroup prebidgroup prebid
USER prebid
EXPOSE 2424
EXPOSE 2525
ENTRYPOINT ["/usr/local/bin/prebid-cache"]
