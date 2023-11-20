# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

all: centos7/generated
centos7/generated: common-generated-all
	test -d centos7/generated || mkdir centos7/generated
	cp -f -rlt centos7/generated common-generated/*

all: debian10/generated
debian10/generated: common-generated-all
	test -d debian10/generated || mkdir debian10/generated
	cp -f -rlt debian10/generated common-generated/*

all: debian11/generated
debian11/generated: common-generated-all
	test -d debian11/generated || mkdir debian11/generated
	cp -f -rlt debian11/generated common-generated/*

all: debian12/generated
debian12/generated: common-generated-all
	test -d debian12/generated || mkdir debian12/generated
	cp -f -rlt debian12/generated common-generated/*

all: rocky8/generated
rocky8/generated: common-generated-all
	test -d rocky8/generated || mkdir rocky8/generated
	cp -f -rlt rocky8/generated common-generated/*

all: ubuntu1804/generated
ubuntu1804/generated: common-generated-all
	test -d ubuntu1804/generated || mkdir ubuntu1804/generated
	cp -f -rlt ubuntu1804/generated common-generated/*

all: ubuntu2004/generated
ubuntu2004/generated: common-generated-all
	test -d ubuntu2004/generated || mkdir ubuntu2004/generated
	cp -f -rlt ubuntu2004/generated common-generated/*

all: ubuntu2204/generated
ubuntu2204/generated: common-generated-all
	test -d ubuntu2204/generated || mkdir ubuntu2204/generated
	cp -f -rlt ubuntu2204/generated common-generated/*

RVMKEY1=mpapis.asc
RVMKEY2=pkuczynski.asc

common-generated-all: common-generated/$(RVMKEY1) common-generated/$(RVMKEY2)

common-generated/$(RVMKEY1): common-generated
	wget -cqO common-generated/$(RVMKEY1) https://rvm.io/mpapis.asc

common-generated/$(RVMKEY2): common-generated
	wget -cqO common-generated/$(RVMKEY2) https://rvm.io/pkuczynski.asc

common-generated:
	mkdir common-generated
