#!/bin/bash

/debezium/conf/configure-connector.sh &

/docker-entrypoint.sh start
