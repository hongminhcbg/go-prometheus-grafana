# RUN PROMETHEUS LOCAL
ref: https://gabrieltanner.org/blog/collecting-prometheus-metrics-in-golang

I. How to run this example
    
step 1: 

    $ docker-compose build && docker-compose up -d

step 2:

    grafana server will be located in http://localhost:3000, defalt username:password is admin:admin

II. Prometheus metric type
    
    Counter: the value can only increase or reset to zero,
        you can use a counter to repesent a number of requests, the task completed
    Gause: a single number like temperature
    Histogram: a samples observations like request duration, hit map for each bucket

