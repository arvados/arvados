#!/bin/sh
while ! psql -c\\du >/dev/null 2>/dev/null ; do
    sleep 1
done
