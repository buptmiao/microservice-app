#!/bin/bash

sudo yum install -y wget git mercurial

wget -q https://github.com/coreos/etcd/releases/download/v3.1.0-rc.1/etcd-v3.1.0-rc.1-linux-amd64.tar.gz
sudo tar -xzf etcd-v3.1.0-rc.1-linux-amd64.tar.gz -C .
sudo cp -a etcd-v3.1.0-rc.1-linux-amd64/{etcd,etcdctl} /usr/local/bin/

wget -q https://storage.googleapis.com/golang/go1.6.2.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.6.2.linux-amd64.tar.gz

echo "export PATH=$PATH:/usr/local/go/bin:/go/bin
export GOPATH=/go
export LOCAL_IP=$LOCAL_IP" >> /home/vagrant/.bash_profile

source /home/vagrant/.bash_profile

go get -u github.com/buptmiao/microservice-app

cd $GOPATH/src/github.com/buptmiao/microservice-app

go get ./...

go install ./...

chown -R vagrant:vagrant /go
