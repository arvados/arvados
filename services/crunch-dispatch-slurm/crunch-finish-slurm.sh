#!/bin/sh

# Script to be called by strigger when a job finishes.  This ensures the job
# record has the correct state "Complete" even if the node running the job
# failed.

ARVADOS_API_HOST=$1
ARVADOS_API_TOKEN=$2
ARVADOS_API_HOST_INSECURE=$3
uuid=$4
jobid=$5

# If it is possible to attach metadata to job records we could look up the
# above information instead of getting it on the command line.  For example,
# this is the recipe for getting the job name (container uuid) from the job id.
#uuid=$(squeue --jobs=$jobid --states=all --format=%j --noheader)

export ARVADOS_API_HOST ARVADOS_API_TOKEN ARVADOS_API_HOST_INSECURE

exec arv container update --uuid $uuid --container '{"state": "Complete"}'
