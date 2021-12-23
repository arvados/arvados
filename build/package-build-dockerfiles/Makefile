# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

SHELL := '/bin/bash'
all: centos7/generated debian10/generated debian11/generated ubuntu1804/generated ubuntu2004/generated

centos7/generated: common-generated-all
	test -d centos7/generated || mkdir centos7/generated
	cp -f -rlt centos7/generated common-generated/*

debian10/generated: common-generated-all
	test -d debian10/generated || mkdir debian10/generated
	cp -f -rlt debian10/generated common-generated/*

debian11/generated: common-generated-all
	test -d debian11/generated || mkdir debian11/generated
	cp -f -rlt debian11/generated common-generated/*

ubuntu1804/generated: common-generated-all
	test -d ubuntu1804/generated || mkdir ubuntu1804/generated
	cp -f -rlt ubuntu1804/generated common-generated/*

ubuntu2004/generated: common-generated-all
	test -d ubuntu2004/generated || mkdir ubuntu2004/generated
	cp -f -rlt ubuntu2004/generated common-generated/*

GOTARBALL_=DOES_NOT_EXIST
NODETARBALL_=DOES_NOT_EXIST
GOTARBALL_x86_64=go1.17.1.linux-amd64.tar.gz
NODETARBALL_x86_64=node-v10.23.1-linux-x64.tar.xz
GOTARBALL_aarch64=go1.17.1.linux-arm64.tar.gz
NODETARBALL_aarch64=node-v10.23.1-linux-arm64.tar.xz

# Get the bash variable $HOSTTYPE (this requires the SHELL line above)
HOSTTYPE=$(shell echo $${HOSTTYPE})

GOTARBALL=${GOTARBALL_$(HOSTTYPE)}
NODETARBALL=${NODETARBALL_$(HOSTTYPE)}

RVMKEY1=mpapis.asc
RVMKEY2=pkuczynski.asc

common-generated-all: common-generated/$(GOTARBALL) common-generated/$(NODETARBALL) common-generated/$(RVMKEY1) common-generated/$(RVMKEY2)

common-generated/$(GOTARBALL): common-generated
	wget -cqO common-generated/$(GOTARBALL) https://dl.google.com/go/$(GOTARBALL)

common-generated/$(NODETARBALL): common-generated
	wget -cqO common-generated/$(NODETARBALL) https://nodejs.org/dist/v10.23.1/$(NODETARBALL)

common-generated/$(RVMKEY1): common-generated
	wget -cqO common-generated/$(RVMKEY1) https://rvm.io/mpapis.asc

common-generated/$(RVMKEY2): common-generated
	wget -cqO common-generated/$(RVMKEY2) https://rvm.io/pkuczynski.asc

common-generated:
	mkdir common-generated
