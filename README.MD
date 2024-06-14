# RonyKIT

RonyKIT is a collection of tools designed to be both extendable and flexible, enabling the rapid development of a complete API/Edge server.
With the RonyKIT framework, you can create API handlers that support both RPC and REST style APIs without the need to rewrite the
endpoint layer multiple times.

If you are seeking a framework to develop your API/Edge server, you can directly use the `rony` package. This package provides an
intuitive framework, leveraging Go Generics to offer a clean and user-friendly API. This is the recommended approach for using
the RonyKIT framework. However, if you require greater control over your API server and have concerns about performance and
memory usage, you can directly utilize the `kit` package, which forms the core of the RonyKIT framework.

- For more information about the `rony` package, please visit [rony](./rony/README.MD).
- For more information about the `kit` package, please visit [kit](./kit/README.MD).

# RonyKIT Standard Packages

This repository contains all projects related to RonyKIT. The central package, which is the core of the RonyKIT framework, is [kit](./kit), with standard modules located in the /std folders.

| Package      | Bundle Type | Version | Description                                                                                                                          |
|--------------|-------------|---------|--------------------------------------------------------------------------------------------------------------------------------------|
| KIT          | -           | v0.15.0 | The main package of the RonyKIT framework                                                                                            |
| fasthttp     | Gateway     | v0.15.0 | The Gateway bundle implemented using the [fasthttp](https://github.com/valyala/fasthttp) framework                                   |
| fastws       | Gateway     | v0.15.0 | The Gateway bundle implemented using [gnet](https://github.com/panjf2000/gnet) and [gobwas](https://github.com/gobwas/ws) frameworks |
| silverhttp   | Gateway     | v0.15.0 | The Gateway bundle implemented using the new super-fast HTTP server [silverlining](https://github.com/go-www/silverlining)           |
| rediscluster | Cluster     | v0.15.0 | The Cluster bundle implemented using [redis](https://github.com/go-redis/redis)                                                      |
| p2pcluster   | Cluster     | v0.15.0 | The Cluster bundle implemented using [p2p](https://github.com/libp2p/go-libp2p)                                                      |
