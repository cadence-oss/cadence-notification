Cadence Notification Service 
----
This is an extension of [Cadence service](https://github.com/uber/cadence) to support notifications. 

See detailed background and proposal in this [issue](https://github.com/uber/cadence/issues/3798).

Running locally with docker-compose
---
Run below command to start local all-in-one docker
```
cd docker/ && docker-compose up 
```
Wait for a few seconds to let everything get stable (no logs rolling).

Then checkout [cadence-samples](https://github.com/uber-common/cadence-samples) to run a helloworld
```
cadence --do samples-domain d re && ./bin/helloworld && ./bin/helloworld -m worker
```  

You will see the test receiver can receive some messages in the logs:
```
cadence-notification_1  | 2022/04/07 04:23:06 [Test server incoming request]: &{0-1  2cab9b7e-40bb-45e2-8438-2c99454f7eaf helloworld_7d1d74ea-4e39-494f-a9cc-7151604ab14a 851dc419-f028-49fa-b10f-9f3d70022d7a main.helloWorldWorkflow 52264409061-03-05 22:26:40 +0000 UTC 1970-01-01 00:00:00 +0000 UTC 1970-01-01 00:00:00 +0000 UTC map[BinaryChecksums:[bb2814702ff3d6522c5b6849d9ade58e] ExecutionTime:0 IsCron:false NumClusters:1 StartTime:1649305385884708000 TaskList:helloWorldGroup WorkflowType:main.helloWorldWorkflow] map[]}, URL: /
cadence-notification_1  | 2022/04/07 04:23:06 [Test server incoming request]: &{0-2  2cab9b7e-40bb-45e2-8438-2c99454f7eaf helloworld_7d1d74ea-4e39-494f-a9cc-7151604ab14a 851dc419-f028-49fa-b10f-9f3d70022d7a main.helloWorldWorkflow 52264409061-03-05 22:26:40 +0000 UTC 1970-01-01 00:00:00 +0000 UTC 1970-01-01 00:00:00 +0000 UTC map[BinaryChecksums:[bb2814702ff3d6522c5b6849d9ade58e] CloseStatus:0 CloseTime:1649305386686346800 ExecutionTime:0 HistoryLength:11 IsCron:false NumClusters:1 StartTime:1649305385884708000 TaskList:helloWorldGroup WorkflowType:main.helloWorldWorkflow] map[]}, URL: /

```

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
