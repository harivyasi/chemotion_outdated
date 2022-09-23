#!/bin/bash

docker compose -f dev.docker-compose.yml run --rm anchor
docker compose -f dev.docker-compose.yml build hull
docker compose -f dev.docker-compose.yml run --rm sail