#!/bin/bash

#!/bin/bash

success=false
max_retries=30
retry_count=0

while true ; do
  response=$(curl -i -X POST -H "Accept: application/json" -H "Content-Type: application/json" http://localhost:8083/connectors/ -d @/debezium/conf/application.json)

  if [[ $response == *"HTTP/1.1 201"* ]] || [[ $response == *"HTTP/1.1 409"* ]]; then
    break
  fi

  sleep 1
done
