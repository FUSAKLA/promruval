FROM golang:1.15 AS builder
COPY . /promruval
WORKDIR /promruval
RUN make build

FROM alpine
COPY --from=builder /promruval/promruval /usr/bin/promruval
ENTRYPOINT ["promruval"]
CMD ["--help"]

