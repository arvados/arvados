all:
	go get .
	docker build --tag=cgofuse --build-arg=http_proxy="$(http_proxy)" --build-arg=https_proxy="$(https_proxy)" "$(GOPATH)"/src/github.com/curoverse/cgofuse
	go get github.com/karalabe/xgo
	xgo --image=cgofuse --targets=linux/amd64,linux/386,darwin/amd64,darwin/386,windows/amd64,windows/386 .
	install arvados-* "$(GOPATH)"/bin/
	rm --interactive=never arvados-*
