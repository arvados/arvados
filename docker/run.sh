#!/bin/sh


echo "Starting documentation server on port 9898"

docker run -d -i -t -p 9898:80 arvados/docserver
