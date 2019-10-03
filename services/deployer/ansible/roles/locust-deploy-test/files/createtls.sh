#!/bin/bash

openssl req \
    -new \
    -newkey rsa:4096 \
    -days 36500 \
    -nodes \
    -x509 \
    -subj "/C=US/ST=CA/L=Malibu/O=My Company/CN=Locust Load Testing" \
    -keyout server.key \
    -out server.crt
