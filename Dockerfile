FROM centos
ADD . /go/src/github.com/buptmiao/msgo
RUN go build -o broker /go/src/github.com/buptmiao/msgo/cmd/broker.go
EXPOSE 13001 13000
ENTRYPOINT ["./broker"]