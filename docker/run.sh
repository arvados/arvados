#!/bin/sh


echo "Starting documentation server: http://localhost:9898"
docker run -d -i -t -p 9898:80 arvados/docserver

echo "Starting api server:     https://localhost:9900"
docker run -d -i -t -p 9900:443 arvados/api

echo "Starting workbench server:     http://localhost:9899"
docker run -d -i -t -p 9899:80 arvados/workbench

echo "Starting sso server:     https://localhost:9901"
docker run -d -i -t -p 9901:443 arvados/sso
