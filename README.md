Cadence Notification Service 
----
This is an extension of [Cadence service](https://github.com/uber/cadence) to support notifications. 

See detailed background and proposal in this [issue](https://github.com/uber/cadence/issues/3798).


Development
----
The default Cadence docker compose files cannot be used. 
Because its Kafka broker DNS are behind docker network and cannot be used by non-docker processes. 
(Please improve it if you know how to workaround)

The following instructions is from [cadence repo](https://github.com/uber/cadence/blob/master/CONTRIBUTING.md). 
#### 1. Install Cadence tools

```
brew install cadence-workflow
``` 
This will install cadence tools including CLI and schema tools.

On Mac, this will also install the schema is at `/usr/local/etc/cadence/schema/`.
You should adjust it accordingly if brew installed the schema in other places. 

#### 2. Start Cadence dependencies: Cassandra+ElasticSearch+Kafka  
```
docker-compose -f ./docker/cassandra-es7-kafka.yml up
```
#### 3. Schema installation 
```
cadence-cassandra-tool create -k cadence --rf 1
cadence-cassandra-tool -k cadence setup-schema -v 0.0
cadence-cassandra-tool -k cadence update-schema -d /usr/local/etc/cadence/schema/cassandra/cadence/versioned
```
will install schema for Cassandra. 
 
```
export ES_SCHEMA_FILE=/usr/local/etc/cadence/schema/schema/elasticsearch/v7/visibility/index_template.json
curl -X PUT "http://127.0.0.1:9200/_template/cadence-visibility-template" -H 'Content-Type: application/json' --data-binary "@$(ES_SCHEMA_FILE)"
curl -X PUT "http://127.0.0.1:9200/cadence-visibility-dev"
```
will install the ElasticSearch schema.

#### 4. 
Production
---
TODO