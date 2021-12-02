Cadence Notification Service 
----
This is an extension of [Cadence service](https://github.com/uber/cadence) to support notifications. 

See detailed background and proposal in this [issue](https://github.com/uber/cadence/issues/3798).

Running locally
---
TODO: add a Dockerfile to build docker image, then make a docker-compose file to run all in one

Running in Production
---
TODO

Development
----
The default Cadence docker compose files cannot be used. 
Because its Kafka broker DNS are behind docker network and cannot be used by non-docker processes. 
(Please improve it if you know how to workaround)

The following instructions is from 
#### 1. Checkout cadence server repo
 Follow the [instruction](https://github.com/uber/cadence/blob/master/CONTRIBUTING.md)
 to start local server binary using `cassandra-es7-kafka.yml`

#### 2. Start a workflow
Use [cadence-samples](https://github.com/uber-common/cadence-samples) to start a helloworld workflow

#### 3.0 Start cadence-notification by command line
```
make bins
```
to generate the `cadence-notification` binary, then
```
./cadence-notification start
```
to start the service.
 
#### 3.1 Alternatively, start with IntelliJ IDE
In IDE, click the run button in the `main.go`

<img width="428" alt="main-run" src="https://user-images.githubusercontent.com/4523955/144361024-259b79db-9f0c-45e1-b1b6-2c1b392b1721.png">
And then `Edit Configurations` to add the `Program Arguments` like below
<img width="1087" alt="ide-config" src="https://user-images.githubusercontent.com/4523955/144361029-cc7e5022-813f-4536-9fe8-0a570e5e16f4.png">
