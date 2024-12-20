FROM golang:1.23 as builder
WORKDIR /go/src
ADD . .

RUN CGO_ENABLED=0 go build -a -buildvcs=false -o /go/bin/mongodb-multicluster

FROM scratch
COPY --from=builder /go/bin/mongodb-multicluster /go/bin/mongodb-multicluster

ENTRYPOINT [ "/go/bin/mongodb-multicluster" ]
