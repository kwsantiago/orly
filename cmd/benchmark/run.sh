#!/usr/bin/env bash

# khatru

khatru &
KHATRU_PID=$!
printf "khatru started pid: %s\n" $KHATRU_PID
sleep 2s
LOG_LEVEL=info relay-benchmark -relay ws://localhost:3334 -events 10000 -queries 100
kill $KHATRU_PID
printf "khatru stopped\n"
sleep 1s

# ORLY

LOG_LEVEL=off \
ORLY_LOG_LEVEL=off \
ORLY_DB_LOG_LEVEL=off \
ORLY_SPIDER_TYPE=none \
ORLY_LISTEN=localhost \
ORLY_PORT=7447 \
ORLY_AUTH_REQUIRED=false \
ORLY_PRIVATE=true \
orly &
ORLY_PID=$!
printf "ORLY started pid: %s\n" $ORLY_PID
sleep 2s
LOG_LEVEL=info relay-benchmark -relay ws://localhost:7447 -events 100 -queries 100
kill $ORLY_PID
printf "ORLY stopped\n"
sleep 1s
