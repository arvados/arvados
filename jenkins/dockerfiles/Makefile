all: centos6/generated debian7/generated ubuntu1204/generated

centos6/generated: common-generated-all
	test -d centos6/generated || mkdir centos6/generated
	cp -rlt centos6/generated common-generated/*

debian7/generated: common-generated-all
	test -d debian7/generated || mkdir debian7/generated
	cp -rlt debian7/generated common-generated/*

ubuntu1204/generated: common-generated-all
	test -d ubuntu1204/generated || mkdir ubuntu1204/generated
	cp -rlt ubuntu1204/generated common-generated/*

common-generated-all: common-generated/golang-amd64.tar.gz

common-generated/golang-amd64.tar.gz: common-generated
	wget -cqO common-generated/golang-amd64.tar.gz https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz

common-generated:
	mkdir common-generated
