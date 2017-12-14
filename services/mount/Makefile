all:
	go get .
	docker build --tag=cgofuse $(GOPATH)/src/github.com/billziss-gh/cgofuse
	go get github.com/karalabe/xgo
	xgo --image=cgofuse --targets=linux/amd64,linux/386,darwin/amd64,darwin/386,windows/amd64,windows/386 .
