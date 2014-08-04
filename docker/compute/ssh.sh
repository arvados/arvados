#!/bin/bash

echo $ENABLE_SSH

# Start ssh daemon if requested via the ENABLE_SSH env variable
if [[ ! "$ENABLE_SSH" =~ (0|false|no|f|^$) ]]; then
echo "STARTING"
  /etc/init.d/ssh start
fi

