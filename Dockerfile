FROM golang:alpine
WORKDIR /
COPY bin/manager /manager
CMD ["/manager"]