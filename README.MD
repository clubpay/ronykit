# RonyKit 
RonyKit provides the abstraction layer for creating a cluster aware API server. By defining separate 
components for each task, you are almost free to develop your API server using RPC (Proto/JSON/...),
REST, GraphQL or even with your in-house defined protocol. However, some components are implemented
in the 'std' which make it easy to use RonyKit out of the box.

## Gateway
Gateway defines the interface that Edge server communicate with clients.

## Cluster
Cluster defines the interface that Edge server instances communicate with each other. 

## Dispatcher
Dispatcher provides the interface to prepare incoming and outgoing messages from the RPC handlers 
to the external world of the system. 

# Benchmark 
## Simple Echo 
We used wrk to benchmark Ronykit standard REST implementation vs Fiber which is fully optimized for REST

RonyKit:
```shell
Running 10s test @ http://127.0.0.1/echo/1200
  12 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     8.22ms    3.15ms  60.00ms   90.54%
    Req/Sec    10.09k     2.04k   18.19k    82.92%
  1208482 requests in 10.05s, 175.18MB read
  Socket errors: connect 0, read 3228, write 0, timeout 0
Requests/sec: 120283.71
Transfer/sec:     17.44MB

```
Fiber:
```shell
  12 threads and 1000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    15.84ms    2.58ms  62.44ms   96.64%
    Req/Sec     5.19k   787.51     5.85k    96.00%
  619345 requests in 10.01s, 73.83MB read
  Socket errors: connect 0, read 3235, write 0, timeout 0
Requests/sec:  61844.21
Transfer/sec:      7.37MB
```

