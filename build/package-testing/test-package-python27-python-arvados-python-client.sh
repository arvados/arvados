#!/bin/sh

exec python2.7 <<EOF
import arvados
print "Successfully imported arvados"
EOF
