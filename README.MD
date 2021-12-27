# RonyKit 
RonyKit provides the abstraction layer for creating a cluster aware API server. By defining separate 
components for each task, you are almost free to develop your API server using RPC (Proto/JSON/...),
REST, GraphQL or even with your in-house defined protocol. However, some components are implemented
in the 'std' which make it easy to use RonyKit out of the box.

## Gateway
## Cluster
## Dispatcher
## Envelope
## Router

# Benchmark 
## Simple Echo 
We used wrk to benchmark Ronykit standard REST implementation vs Fiber which is fully optimized for REST

RonyKit:
```shell
12 threads and 1000 connections
Thread Stats   Avg      Stdev     Max   +/- Stdev
Latency    15.74ms    2.68ms  65.92ms   98.11%
Req/Sec     5.22k   781.12     5.87k    95.58%
623619 requests in 10.01s, 90.40MB read
Socket errors: connect 0, read 3270, write 0, timeout 0
Requests/sec:  62276.11
Transfer/sec:      9.03MB
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

