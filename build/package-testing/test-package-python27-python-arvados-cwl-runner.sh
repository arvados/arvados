#!/bin/sh

exec python <<EOF
import arvados_cwl
print "arvados-cwl-runner version", arvados_cwl.__version__
EOF
