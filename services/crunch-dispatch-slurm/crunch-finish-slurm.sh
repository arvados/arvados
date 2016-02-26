#!/bin/sh

# I wonder if it is possible to attach metadata to job records to look these
# things up instead of having to provide it on the command line.

ARVADOS_API_HOST=$1
ARVADOS_API_TOKEN=$2
ARVADOS_API_HOST_INSECURE=$3
uuid=$4
jobid=$5

#uuid=$(squeue --jobs=$jobid --states=all --format=%j --noheader)

export ARVADOS_API_HOST ARVADOS_API_TOKEN ARVADOS_API_HOST_INSECURE

exec arv container update --uuid $uuid --container '{"state": "Complete"}'
