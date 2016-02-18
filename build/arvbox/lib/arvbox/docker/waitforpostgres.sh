#!/bin/sh
while ! psql postgres -c\\du >/dev/null 2>/dev/null ; do
    sleep 1
done
