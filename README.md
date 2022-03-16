# 基于grpc的微服务客户端和服务端

- 微服务客户端和服务端都是用原生grpc实现
- 在grpc server加上了微服务的注册代码
- 在grpc client加上了微服务的发现代码
- 服务发现与注册提供了etcdv3的实现，使用接口，可自己替换
- 调用负载均衡方案是客户端负载均衡，使用接口，可自己替换
- 默认负载均衡支持四种策略：轮询、随机、一致性哈希、指定地址
- grpc client服务发现与负载均衡不使用grpc官方api，因为官方api太难用了，花里胡哨的

[server example](example/server.go)

[client example](example/client.go)
