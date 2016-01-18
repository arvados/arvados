#!/bin/sh

exec python <<EOF
import arvados
print "Successfully imported arvados"
EOF
