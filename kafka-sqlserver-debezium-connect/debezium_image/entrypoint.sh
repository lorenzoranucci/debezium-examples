#!/bin/bash

echo "Starting 'configure-connector' background task"

/debezium/configure-connector.sh &

echo "Starting 'docker-entrypoint'"

/docker-entrypoint.sh start
