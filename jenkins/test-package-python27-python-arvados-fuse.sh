#!/bin/sh

exec python <<EOF
import arvados_fuse
print "Successly imported arvados_fuse"
EOF
