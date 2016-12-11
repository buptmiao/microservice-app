## Microservice-app

### 一. 简介

该项目是基于go语言搭建的微服务架构应用. 包含如下组件:   

1. 服务注册中心 [etcd](https://github.com/coreos/etcd)  
2. Api 网关  
3. Feed 服务  
4. Profile 服务  
5. Topic 服务  
6. 监控组件: prometheus + grafana  

其中Feed, Profile, Topic 启动时会向etcd注册服务, Apigateway 通过调用这三个服务的客户端 Watch 到相应服务的注册Key, 同时得到服务的地址. 当服务实例个数动态伸缩时, Apigateway 也会实时响应变化.

结构如下:  

![block](https://github.com/buptmiao/microservice-app/blob/master/pictures/block.png) 

### 二. 项目源码

目录 | 介绍 
--------|-----------------
apigateway  |  注册app所有endpoint.
client      |  所有访问微服务的客户端, 供apigateway调用. 提供服务发现,负载均衡,错误重试和故障降级等功能.
cmd         |  各个服务的启动命令.
docker      |  构建各个服务的docker镜像.
feed        |  feed服务.
monitor     |  监控组件.
profile     |  profile服务.
proto       |  服务间IPC方式采用grpc.
topic       |  topic服务.
vagrant     |  虚拟化分布式环境, 采用传统方式部署应用.
    
### 三. 部署应用

目前使用了两种应用部署方式:传统部署方式和容器化部署方式

#### 1. 传统部署
 如果你熟悉[vagrant](https://www.vagrantup.com/), vagrant目录下有具体部署细节. 参考[Vagrantfile](https://github.com/buptmiao/microservice-app/blob/master/vagrant/Vagrantfile) 和 [provision.sh](https://github.com/buptmiao/microservice-app/blob/master/vagrant/provision.sh)
 总的来讲,项目使用vagrant虚拟化了5个节点, 节点0部署etcd, 节点1-4分别部署service-feed, service-profile, service-topic, apigateway.
```ruby
 $nodes = 5
 Vagrant.configure("2") do |config|
     config.vm.box = "centos/7"
     (0..($nodes - 1)).each do |i|
         config.vm.define name="node-#{i}", primary: (i == 0), autostart: (i == 0) do |node|
             node.vm.hostname = name
             node.vm.network "private_network", ip: "192.168.50.#{10+i}"
             node.vm.provision "shell", path: "provision.sh", env: {"LOCAL_IP" => "192.168.50.#{10+i}", "ETCD_ENDPOINT" => "http://192.168.50.10:2379"}
         end
     end
 end
```

部署前请确保[vagrant-1.9.0](https://releases.hashicorp.com/vagrant/1.9.0/), 至于为什么是该版本, 个人认为该版本目前(2016-12-10)来看最稳定,bug最少.

在vagrant目录下, 使用如下命令, 启动所有节点. 该命令第一次启动时会创建5台虚拟机node-0 ~ node-4. 并下载安装所需的可执行文件.
```
# 注意: 首次启动会比较慢, 具体时间取决于网络.
$ vagrant up /node-./
```
启动后, 可以通过`vagrant ssh` + `node-*` 连接任何虚拟机. 例如,如果想查看apigateway是否运行, 可以执行如下命令:
```
$ vagrant ssh node-4
$ ps -ef | grep apigateway
$ exit
```

如果启动成功, 那么可以访问我们的服务了

```
$ curl -XPUT "http://192.168.50.14:8080/api/feed/create_feed" -d '{"id": 100, "user_id": 123, "content": "hello world"}'  // 发布feed1
$ curl -XPUT "http://192.168.50.14:8080/api/feed/create_feed" -d '{"id": 101, "user_id": 123, "content": "goodbye!"}'     // 发布feed2
$ curl -XGET "http://192.168.50.14:8080/api/feed/get_feeds?user_id=123&&size=2"                                           // 拉取feed列表
```

将会显示
```
{
    "feeds": [
        {
            "id": 100,
            "user_id": 123,
            "content": "hello world"
        },
        {
            "id": 101,
            "user_id": 123,
            "content": "goodbye!"
        }
    ]
}
```

注意: 默认每一个微服务只启动一个实例, 如果想看多个微服务实例, 那么可以到某个节点上手动启动. 例如:
```
$ vagrant ssh node-2
$ nohup feed -addr=$LOCAL_IP:8082 -etcd.addr=$ETCD_ENDPOINT 0<&- &>/dev/null &  //nohup 忽略用户退出时的hup信号, 这样当退出ssh时feed进程不会受到影响. 实际上feed进程源码中实现了对某些信号的处理.
$ exit
```
这样, 对于feed相关的请求,apigateway会把每一个请求通过round robin的方式均衡的打到两个feed实例上,实现进程内负载均衡. 同样需要注意: 原则上说, 微服务都应该是无状态的. 然而为了简单,该项目中的微服务实例都是采用内存存储. 所以在多实例环境下, 如果你发布了一条feed, 却没有拉取到, 那么多试几次即可.

#### 2. 容器化部署

如果你对docker熟悉的话, docker目录下提供了构建镜像的脚本 [build.sh](https://github.com/buptmiao/microservice-app/blob/master/docker/build.sh).
```
./build.sh
```
该脚本生成4个服务的docker镜像, 然后我们通过docker-compose命令启动容器.

```
docker-compose up -d
```

启动成功后:
```
$ curl -XPUT "http://localhost:8080/api/feed/create_feed" -d '{"id": 100, "user_id": 123, "content": "hello world"}'  // 发布feed1
$ curl -XPUT "http://localhost:8080/api/feed/create_feed" -d '{"id": 101, "user_id": 123, "content": "goodbye!"}'     // 发布feed2
$ curl -XGET "http://localhost:8080/api/feed/get_feeds?user_id=123&&size=2"                                           // 拉取feed列表
```

### 四. 应用监控

应用监控采用[prometheus](https://github.com/prometheus/prometheus) + [grafana](https://github.com/grafana/grafana) + [cadvisor](https://github.com/google/cadvisor) + [alertmanager](https://github.com/prometheus/alertmanager).

#### 启动监视器

启动监视器之前请先阅读[README](https://github.com/buptmiao/microservice-app/blob/master/monitor/README.md). 

如果是使用方式1部署的应用, 在monitor目录下, 可以通过如下配置target, 来监视app

```
- targets: ['localhost:9090','cadvisor:8080', '192.168.50.11:6062', '192.168.50.12:6063', '192.168.50.13:6064', '192.168.50.14:6060']
```

然后在monitor目录下
```
$ docker-compose up
```

这样监视器启动成功.

如果采用方式2部署的应用, docker-compose文件已经配置好. 直接在monitor目录下:
```
$ docker-compose -f docker-compose.yml.2 up -d
```

两种方式启动成功后, 都可以通过访问: http://localhost:9090/graph 来查看metrics.

#### 可视化

可视化采用grafana, 它与prometheus结合的很好, 采用该方案可以很好的监控docker容器的状态

打开浏览器 http://localhost:3000, 进入grafana, 添加数据源, Type选择Prometheus, Access选择direct模式, 填写prometheus的url: http://localhost:9090, 勾上默认. Save & test. 退出.

添加dashboard, 导入monitor/grafana/docker_dashboard.json 即可看到下图:

![docker_dashboard](https://github.com/buptmiao/microservice-app/blob/master/pictures/docker_dashboard.png) 

### 五. Todo

* 使用zipkin跟踪
* 使用kubenetes部署整个应用
