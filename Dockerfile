FROM golang:1.15.0-buster AS builder-go
ENV GO111MODULE=on
WORKDIR /blooming-password
COPY . /blooming-password
RUN make build-go

FROM debian:latest AS builder-filter
WORKDIR /blooming-password
COPY --from=builder-go /blooming-password/configs/blooming-password-filter-create.conf configs/
COPY --from=builder-go /blooming-password/tools/blooming-password-filter-create tools/
COPY --from=builder-go /blooming-password/Makefile .
RUN apt-get update -y && apt-get install -y make
RUN make build-filter

FROM gcr.io/distroless/base:nonroot
VOLUME /etc/ssl/
COPY --from=builder-go /blooming-password/configs/blooming-password-server.conf configs/
COPY --from=builder-go /blooming-password/bin/blooming-password-server bin/
COPY --from=builder-filter /blooming-password/data/1-16-pwned-passwords-sha1-ordered-by-count-v6.filter data/
EXPOSE 9379
ENTRYPOINT [ "bin/blooming-password-server", "--config", "configs/blooming-password-server.conf" ]
