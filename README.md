Cadence Notification Service 
----
This is an extension of [Cadence service](https://github.com/uber/cadence) to support notifications. 

See detailed background and proposal in this [issue](https://github.com/uber/cadence/issues/3798).

Running locally
---
TODO 

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

#### 3. Start cadence-notification
```

```