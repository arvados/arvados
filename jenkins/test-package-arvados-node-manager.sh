#!/bin/sh
exec python <<EOF
import libcloud.compute.types
import libcloud.compute.providers
libcloud.compute.providers.get_driver(libcloud.compute.types.Provider.AZURE_ARM)
print "Successly imported compatible libcloud library"
EOF
