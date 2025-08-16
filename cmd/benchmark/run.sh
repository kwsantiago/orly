#!/usr/bin/env bash
docker compose up -d --remove-orphans
docker compose run benchmark -relay ws://orly:7447 -events 10000 -queries 100
docker compose run benchmark -relay ws://khatru:7447 -events 10000 -queries 100
docker compose run benchmark -relay ws://strfry:7777 -events 10000 -queries 100
docker compose run benchmark -relay ws://relayer:7447 -events 10000 -queries 100
