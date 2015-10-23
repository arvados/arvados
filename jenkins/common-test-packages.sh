#!/bin/sh

FAIL=0

echo
for so in $(find . -name "*.so") ; do
    if ldd $so | grep "not found" ; then
        echo "^^^ Missing while scanning $so ^^^"
        FAIL=1
    fi
done

echo
if ! python <<EOF
import arvados
import arvados_fuse
print "Successly imported arvados and arvados_fuse"

import libcloud.compute.types
import libcloud.compute.providers
libcloud.compute.providers.get_driver(libcloud.compute.types.Provider.AZURE_ARM)
print "Successly imported compatible libcloud library"
EOF
then
    FAIL=1
fi

exit $FAIL
