# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

SHELL := '/bin/bash'

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

all: ubuntu2004/generated
ubuntu2004/generated: common-generated-all
	test -d ubuntu2004/generated || mkdir ubuntu2004/generated
	cp -f -rlt ubuntu2004/generated common-generated/*

all: ubuntu2204/generated
ubuntu2204/generated: common-generated-all
	test -d ubuntu2204/generated || mkdir ubuntu2204/generated
	cp -f -rlt ubuntu2204/generated common-generated/*

GOTARBALL_=DOES_NOT_EXIST
NODETARBALL_=DOES_NOT_EXIST
GOVERSION=$(shell grep 'const goversion =' ../../lib/install/deps.go |awk -F'"' '{print $$2}')
GOTARBALL_x86_64=go$(GOVERSION).linux-amd64.tar.gz
NODETARBALL_x86_64=node-v12.22.12-linux-x64.tar.xz
GOTARBALL_aarch64=go$(GOVERSION).linux-arm64.tar.gz
NODETARBALL_aarch64=node-v12.22.12-linux-arm64.tar.xz

# Get the bash variable $HOSTTYPE (this requires the SHELL line above)
HOSTTYPE=$(shell echo $${HOSTTYPE})

GOTARBALL=${GOTARBALL_$(HOSTTYPE)}
NODETARBALL=${NODETARBALL_$(HOSTTYPE)}

common-generated-all: common-generated/$(GOTARBALL) common-generated/$(NODETARBALL)

common-generated/$(GOTARBALL): common-generated
	wget -cqO common-generated/$(GOTARBALL) https://dl.google.com/go/$(GOTARBALL)

common-generated/$(NODETARBALL): common-generated
	wget -cqO common-generated/$(NODETARBALL) https://nodejs.org/dist/v12.22.12/$(NODETARBALL)

common-generated:
	mkdir common-generated
