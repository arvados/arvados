#! /bin/bash

# make sure Ruby is installed before proceeding
if ! ruby -v > /dev/null 2>&1
then
    echo "Installing Ruby. You may be required to enter your password."
    sudo apt-get update
    sudo apt-get install ruby
fi

./build.rb $*
