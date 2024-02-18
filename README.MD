# RonyKIT

RonyKIT is a set of tools that are designed to be extendable and flexible to write a complete API/Edge server quickly.
When you develop API handlers using RonyKIT framework, you can support RPC and REST style APIs without re-writing the
endpoint layer multiple times.

If you are looking for a framework to write your API/Edge server, you can use `rony` package directly, which provides
an easy-to-use framework and with help of Go Generics, it provides a very clean and easy to use API. This is the
recommended way to use RonyKIT framework. However, if you want to have more control over your API server, and are
very concern about performance and memory usage, you can use `kit` package directly, which is the heart of the RonyKIT
framework.

- For more information about `rony` package, please visit [rony](./rony/README.MD)
- For more information about `kit` package, please visit [kit](./kit/README.MD)

# RonyKIT Standard Packages

This repository is holding all the projects related to RonyKIT. The main package which is the heart of
the RonyKIT framework is [kit](./kit) and the standard modules are in the /std folders.

| Package      | BundleType | Version  | Description                                                                                                                          |
|--------------|------------|----------|--------------------------------------------------------------------------------------------------------------------------------------|
| KIT          | -          | v0.11.40 | the main package of the RonyKIT framework                                                                                            |
| fasthttp     | Gateway    | v0.11.40 | the Gateway bundle implemented using [fasthttp](https://github.com/valyala/fasthttp) framework                                       |
| fastws       | Gateway    | v0.11.40 | the Gateway bundle implemented using [gnet](https://github.com/panjf2000/gnet) and [gobwas](https://github.com/gobwas/ws) frameworks |
| silverhttp   | Gateway    | v0.11.40 | the Gateway bundle implemented using the new kid super-fast http server [silverlining](https://github.com/go-www/silverlining)       |
| rediscluster | Cluster    | v0.11.40 | the Cluster bundle implemented using [redis](https://github.com/go-redis/redis)                                                      |
| p2pcluster   | Cluster    | v0.11.40 | the Cluster bundle implemented using [p2p](https://github.com/libp2p/go-libp2p)                                                      |                                                                                                                     |

