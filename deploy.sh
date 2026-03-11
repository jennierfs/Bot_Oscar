#!/bin/bash

docker-compose down -v
docker system prune -af
docker-compose up -d
