all: centos6/generated centos7/generated debian7/generated debian8/generated ubuntu1204/generated ubuntu1404/generated

centos6/generated: common-generated-all
	test -d centos6/generated || mkdir centos6/generated
	cp -rlt centos6/generated common-generated/*

centos7/generated: common-generated-all
	test -d centos7/generated || mkdir centos7/generated
	cp -rlt centos7/generated common-generated/*

debian7/generated: common-generated-all
	test -d debian7/generated || mkdir debian7/generated
	cp -rlt debian7/generated common-generated/*

debian8/generated: common-generated-all
	test -d debian8/generated || mkdir debian8/generated
	cp -rlt debian8/generated common-generated/*

ubuntu1204/generated: common-generated-all
	test -d ubuntu1204/generated || mkdir ubuntu1204/generated
	cp -rlt ubuntu1204/generated common-generated/*

ubuntu1404/generated: common-generated-all
	test -d ubuntu1404/generated || mkdir ubuntu1404/generated
	cp -rlt ubuntu1404/generated common-generated/*

GOTARBALL=go1.6.2.linux-amd64.tar.gz

common-generated-all: common-generated/$(GOTARBALL)

common-generated/$(GOTARBALL): common-generated
	wget -cqO common-generated/$(GOTARBALL) http://storage.googleapis.com/golang/$(GOTARBALL)

common-generated:
	mkdir common-generated
