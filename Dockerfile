FROM golang:stretch

ENV GOPATH /go
ENV APPPATH $GOPATH/src/github.com/lovoo/ipmi_exporter

COPY . $APPPATH
RUN apt-get update && \
    apt-get install -y freeipmi-tools && \
    cd $APPPATH && make build && mv build/ipmi_exporter/ipmi_exporter / && \
    rm -rf $GOPATH

EXPOSE 9289

ENTRYPOINT [ "/ipmi_exporter" ]
