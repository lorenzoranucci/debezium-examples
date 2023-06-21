#!/bin/bash

# This script is taken form https://hub.docker.com/r/debezium/server

docker stop $(docker ps -aq) && docker container prune -f

docker run -d --name postgres -p 5432:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres debezium/example-postgres

docker run -d --name pulsar -p 6650:6650 -p 7080:8080 apachepulsar/pulsar:3.0.0 bin/pulsar standalone

docker logs -f pulsar

#mkdir {data,conf}; chmod 777 {data,conf}
#cat <<-EOF > conf/application.properties
#debezium.sink.type=pulsar
#debezium.sink.pulsar.client.serviceUrl=pulsar://pulsar:6650
#debezium.source.connector.class=io.debezium.connector.postgresql.PostgresConnector
#debezium.source.offset.storage.file.filename=data/offsets.dat
#debezium.source.offset.flush.interval.ms=0
#debezium.source.database.hostname=postgres
#debezium.source.database.port=5432
#debezium.source.database.user=postgres
#debezium.source.database.password=postgres
#debezium.source.database.dbname=postgres
#debezium.source.database.server.name=tutorial
#debezium.source.schema.whitelist=inventory
#debezium.source.plugin.name=pgoutput
#debezium.source.topic.prefix=sample_prefix_
#EOF


docker run -p 8080:8080 -v $PWD/conf:/debezium/conf -v $PWD/data:/debezium/data --link postgres --link pulsar debezium/server:2.3
wget https://archive.apache.org/dist/pulsar/pulsar-2.9.1/apache-pulsar-2.9.1-bin.tar.gz
tar xvfz ./apache-pulsar-2.9.1-bin.tar.gz

apache-pulsar-2.9.1/bin/pulsar-admin --admin-url http://localhost:7080 topics list "public/default"

apache-pulsar-2.9.1/bin/pulsar-admin --admin-url http://localhost:7080 topics examine-messages --initialPosition latest --messagePosition 1 "persistent://public/default/sample_prefix_.inventory.customers"

