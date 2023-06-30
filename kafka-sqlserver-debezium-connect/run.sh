#!/bin/bash

docker stop $(docker ps -aq) && docker container prune -f

docker compose up -d
