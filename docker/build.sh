#! /bin/bash

# make sure Ruby 1.9.3 is installed before proceeding
if ! ruby -v 2>/dev/null | grep '1\.9\.3' > /dev/null
then
    echo "Installing Ruby. You may be required to enter your password."
    sudo apt-get update
    sudo apt-get -y install ruby1.9.3
fi

./build.rb $*
