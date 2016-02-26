#!/bin/sh

jobid=$1

uuid=$(squeue --jobs=$jobid --states=all --format=%j --noheader)

arv containers update --uuid $uuid --container '{"state": "Completed"}'
