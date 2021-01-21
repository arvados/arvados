# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

all: centos7/generated debian9/generated debian10/generated ubuntu1604/generated ubuntu1804/generated

centos7/generated: common-generated-all
	test -d centos7/generated || mkdir centos7/generated
	cp -f -rlt centos7/generated common-generated/*

debian9/generated: common-generated-all
	test -d debian9/generated || mkdir debian9/generated
	cp -f -rlt debian9/generated common-generated/*

debian10/generated: common-generated-all
	test -d debian10/generated || mkdir debian10/generated
	cp -f -rlt debian10/generated common-generated/*


ubuntu1604/generated: common-generated-all
	test -d ubuntu1604/generated || mkdir ubuntu1604/generated
	cp -f -rlt ubuntu1604/generated common-generated/*

ubuntu1804/generated: common-generated-all
	test -d ubuntu1804/generated || mkdir ubuntu1804/generated
	cp -f -rlt ubuntu1804/generated common-generated/*

GOTARBALL=go1.13.4.linux-amd64.tar.gz
NODETARBALL=node-v10.23.1-linux-x64.tar.xz
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
