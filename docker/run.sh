#!/bin/sh


echo "Starting documentation server: http://localhost:9898"
docker run -d -i -t -p 9898:80 arvados/docserver

echo "Starting workbench server:     http://localhost:9899"
docker run -d -i -t -p 9899:80 arvados/workbench
