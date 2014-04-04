#! /bin/sh

# Install prerequisites.
sudo apt-get install curl libcurl3 libcurl3-gnutls libcurl4-openssl-dev python-pip

# Install RVM.
curl -sSL https://get.rvm.io | bash -s stable
source ~/.rvm/scripts/rvm
rvm install 2.1.0

# Install arvados-cli.
gem install arvados-cli
sudo pip install --upgrade httplib2
