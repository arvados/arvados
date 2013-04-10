---
layout: default
navsection: userguide
title: Command line SDK
navorder: 10
---

# Command line SDK

If you are logged in to an Arvados VM, the command line SDK is
probably already installed. Try:

    arv --help

To install:

    echo "deb http://apt.arvados.org/apt precise main contrib non-free" \
      | sudo tee -a /etc/apt/sources.list.d/arvados.list
    wget -q http://apt.arvados.org/53212765.key -O- \
      | sudo apt-key add -
    sudo apt-get update
    sudo apt-get install arvados-cli
