# RUN PROMETHEUS LOCAL
ref: https://gabrieltanner.org/blog/collecting-prometheus-metrics-in-golang

I. How to run this example
    
step 1: 

    $ docker-compose build && docker-compose up -d

step 2:

    grafana server will be located in http://localhost:3000, defalt username:password is admin:admin
    Add new data source http://prometheus:9090 and test

II. Prometheus metric type
    
    Counter: the value can only increase or reset to zero,
        you can use a counter to repesent a number of requests, the task completed
    Gause: a single number like temperature
    Histogram: a samples observations like request duration, hit map for each bucket

III. Demo grafana

![alt text](https://github.com/hongminhcbg/go-prometheus-grafana/blob/main/imgs/demo.jpeg/?raw=true)

![alt text](./imgs/histogram.jpeg/?raw=true)


