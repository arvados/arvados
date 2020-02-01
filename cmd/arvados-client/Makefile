# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

all:
	go mod download
	docker build --tag=cgofuse --build-arg=http_proxy="$(http_proxy)" --build-arg=https_proxy="$(https_proxy)" https://github.com/arvados/cgofuse.git
	go run github.com/karalabe/xgo --image=cgofuse --targets=linux/amd64,linux/386,darwin/amd64,darwin/386,windows/amd64,windows/386 .
	install arvados-* "$(GOPATH)"/bin/
	rm --interactive=never arvados-*
