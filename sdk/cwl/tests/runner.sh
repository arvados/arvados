#!/bin/sh
exec arvados-cwl-runner --disable-reuse --compute-checksum "$@"
