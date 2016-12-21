#!/bin/sh
while ! pg_isready >/dev/null; do
    sleep 0.2
done
