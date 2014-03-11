#! /bin/bash

# make sure Ruby 1.9.3 is installed before proceeding
if ! ruby -e 'exit RUBY_VERSION >= "1.9.3"' 2>/dev/null
then
    echo "Installing Arvados requires at least Ruby 1.9.3."
    echo "You may need to enter your password."
    read -p "Press Ctrl-C to abort, otherwise we will install ruby1.9.3 now..." unused
    
    sudo apt-get update
    sudo apt-get -y install ruby1.9.3
fi

./build.rb $*
