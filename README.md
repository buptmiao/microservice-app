## Microservice-app

该项目是基于go语言搭建的微服务架构应用. 包含5个组件: 
1. 服务注册中心
2. Api 网关
3. Feed 服务
4. Profile 服务
5. Topic 服务
6. 监控组件: prometheus + grafana

其中Feed Profile Topic 启动时会向etcd注册服务, Apigateway 通过调用这三个服务的客户端 Watch 到相应服务的注册Key, 同时得到服务的地址. 当服务实例个数动态伸缩时, Apigateway 也会实时响应变化.

[vagrant-1.9.0](https://releases.hashicorp.com/vagrant/1.9.0/)

