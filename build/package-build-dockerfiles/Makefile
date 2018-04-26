# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

all: centos7/generated debian8/generated debian9/generated ubuntu1204/generated ubuntu1404/generated ubuntu1604/generated

centos7/generated: common-generated-all
	test -d centos7/generated || mkdir centos7/generated
	cp -rlt centos7/generated common-generated/*

debian8/generated: common-generated-all
	test -d debian8/generated || mkdir debian8/generated
	cp -rlt debian8/generated common-generated/*

debian9/generated: common-generated-all
	test -d debian9/generated || mkdir debian9/generated
	cp -rlt debian9/generated common-generated/*

ubuntu1204/generated: common-generated-all
	test -d ubuntu1204/generated || mkdir ubuntu1204/generated
	cp -rlt ubuntu1204/generated common-generated/*

ubuntu1404/generated: common-generated-all
	test -d ubuntu1404/generated || mkdir ubuntu1404/generated
	cp -rlt ubuntu1404/generated common-generated/*

ubuntu1604/generated: common-generated-all
	test -d ubuntu1604/generated || mkdir ubuntu1604/generated
	cp -rlt ubuntu1604/generated common-generated/*

GOTARBALL=go1.10.1.linux-amd64.tar.gz
NODETARBALL=node-v6.11.2-linux-x64.tar.xz

common-generated-all: common-generated/$(GOTARBALL) common-generated/$(NODETARBALL)

common-generated/$(GOTARBALL): common-generated
	wget -cqO common-generated/$(GOTARBALL) http://storage.googleapis.com/golang/$(GOTARBALL)

common-generated/$(NODETARBALL): common-generated
	wget -cqO common-generated/$(NODETARBALL) https://nodejs.org/dist/v6.11.2/$(NODETARBALL)

common-generated:
	mkdir common-generated
