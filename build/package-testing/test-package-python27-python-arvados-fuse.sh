#!/bin/sh

exec python <<EOF
import arvados_fuse
print "Successfully imported arvados_fuse"
EOF
