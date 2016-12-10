#!/bin/bash

sudo yum install -y wget git mercurial

if [ `hostname` == "node-0" ];then
    echo "Installing etcd binaries..."
    [ ! -f "etcd-v3.0.0-linux-amd64.tar.gz" ] && wget -q https://github.com/coreos/etcd/releases/download/v3.0.0/etcd-v3.0.0-linux-amd64.tar.gz
    sudo tar -xzf etcd-v3.0.0-linux-amd64.tar.gz -C .
    sudo cp -a etcd-v3.0.0-linux-amd64/{etcd,etcdctl} /usr/local/bin/
else
    echo "Installing micro binaries..."
    [ ! -f "microservice-app-v1.0.1-linux-amd64.tar.gz" ] && wget -q https://github.com/buptmiao/microservice-app/releases/download/v1.0.1/microservice-app-v1.0.1-linux-amd64.tar.gz
    sudo tar -xzf microservice-app-v1.0.1-linux-amd64.tar.gz -C .
    sudo cp -a microservice-app-v1.0.1-linux-amd64/* /usr/local/bin/
fi

echo "export PATH=$PATH:/usr/local/bin
export LOCAL_IP=$LOCAL_IP
export ETCD_ENDPOINT=$ETCD_ENDPOINT" >> /home/vagrant/.bash_profile

source /home/vagrant/.bash_profile

#go get -u github.com/buptmiao/microservice-app
#
#cd $GOPATH/src/github.com/buptmiao/microservice-app
#
#go get ./...
#
#go install ./...
#
#chown -R vagrant:vagrant /go

# start up services
case `hostname` in
    "node-0")
        etcd --listen-client-urls $ETCD_ENDPOINT --advertise-client-urls $ETCD_ENDPOINT &
        echo "Start up etcd at $ETCD_ENDPOINT."
    ;;
    "node-1")
        nohup feed -addr=$LOCAL_IP:8082 -etcd.addr=$ETCD_ENDPOINT 0<&- &>/dev/null &
        echo "Start up feed at $LOCAL_IP:8082..."
    ;;
    "node-2")
        nohup profile -addr=$LOCAL_IP:8083 -etcd.addr=$ETCD_ENDPOINT 0<&- &>/dev/null &
        echo "Start up profile at $LOCAL_IP:8083..."
    ;;
    "node-3")
        nohup topic -addr=$LOCAL_IP:8084 -etcd.addr=$ETCD_ENDPOINT 0<&- &>/dev/null &
        echo "Start up topic at $LOCAL_IP:8084..."
    ;;
    "node-4")
        nohup apigateway -http.addr=$LOCAL_IP:8080 -etcd.addr=$ETCD_ENDPOINT 0<&- &>/dev/null &
        echo "Start up apigateway at $LOCAL_IP:8080..."
    ;;
esac
