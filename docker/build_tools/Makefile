# This is the 'shell hack'. Call make with DUMP=1 to see the effect.
ifdef DUMP
OLD_SHELL := $(SHELL)
SHELL = $(warning [$@])$(OLD_SHELL) -x
endif

all: skydns-image skydock-image api-image compute-image doc-image workbench-image keep-image keep-proxy-image sso-image shell-image

IMAGE_FILES := $(shell ls *-image 2>/dev/null |grep -v -E 'debian-arvados-image|skydns-image|skydock-image')
GENERATED_DIRS := $(shell ls */generated 2>/dev/null)

# `make clean' removes the files generated in the build directory
# but does not remove any docker images generated in previous builds
clean:
	@echo "make clean"
	-@rm -rf build
	+@[ "$(IMAGE_FILES)" = "" ] || rm -f $(IMAGE_FILES) 2>/dev/null
	+@[ "$(GENERATED_DIRS)" = "" ] || rm -rf */generated 2>/dev/null

DEBIAN_IMAGE := $(shell $(DOCKER) images -q arvados/debian |head -n1)

REALCLEAN_CONTAINERS := $(shell $(DOCKER) ps -a |grep -e arvados -e api_server -e keep_server -e keep_proxy_server -e doc_server -e workbench_server |cut -f 1 -d' ')
# Generate a list of docker images tagged as arvados/*
# but exclude those tagged as arvados/build
ADI_TEMPFILE := $(shell mktemp)
ARVADOS_DOCKER_IMAGES := $(shell $(DOCKER) images -q arvados/* |sort > $(ADI_TEMPFILE))
ABDI_TEMPFILE := $(shell mktemp)
ARVADOS_BUILD_DOCKER_IMAGES := $(shell $(DOCKER) images -q arvados/build |sort > $(ABDI_TEMPFILE))
REALCLEAN_IMAGES := $(shell comm -3 $(ADI_TEMPFILE) $(ABDI_TEMPFILE) |grep -v $(DEBIAN_IMAGE) 2>/dev/null)
DEEPCLEAN_IMAGES := $(shell comm -3 $(ADI_TEMPFILE) $(ABDI_TEMPFILE))
SKYDNS_CONTAINERS := $(shell $(DOCKER) ps -a |grep -e crosbymichael/skydns -e crosbymichael/skydock |cut -f 1 -d' ')
SKYDNS_IMAGES := $(shell $(DOCKER) images -q crosbymichael/skyd*)

# `make realclean' will also remove the Arvados docker images (but not the
# arvados/debian image) and force subsequent makes to build the entire chain
# from the ground up
realclean: clean
	@echo "make realclean"
	+@[ "`$(DOCKER) ps -q`" = '' ] || $(DOCKER) stop `$(DOCKER) ps -q`
	+@[ "$(REALCLEAN_CONTAINERS)" = '' ] || $(DOCKER) rm $(REALCLEAN_CONTAINERS)
	+@[ "$(REALCLEAN_IMAGES)" = '' ] || $(DOCKER) rmi $(REALCLEAN_IMAGES)

# `make deepclean' will remove all Arvados docker images and the skydns/skydock
# images and force subsequent makes to build the entire chain from the ground up
deepclean: clean
	@echo "make deepclean"
	-@rm -f debian-arvados-image 2>/dev/null
	-@rm -f skydns-image skydock-image 2>/dev/null
	+@[ "`$(DOCKER) ps -q`" = '' ] || $(DOCKER) stop `$(DOCKER) ps -q`
	+@[ "$(REALCLEAN_CONTAINERS)" = '' ] || $(DOCKER) rm $(REALCLEAN_CONTAINERS)
	+@[ "$(DEEPCLEAN_IMAGES)" = '' ] || $(DOCKER) rmi $(DEEPCLEAN_IMAGES)
	+@[ "$(SKYDNS_CONTAINERS)" = '' ] || $(DOCKER) rm $(SKYDNS_CONTAINERS)
	+@[ "$(SKYDNS_IMAGES)" = '' ] || $(DOCKER) rmi $(SKYDNS_IMAGES)

# ============================================================
# Dependencies for */generated files which are prerequisites
# for building docker images.

CONFIG_RB = build_tools/config.rb

BUILD = build/.buildstamp

BASE_DEPS = base/Dockerfile config.yml $(BASE_GENERATED)

SLURM_DEPS = slurm/Dockerfile config.yml $(SLURM_GENERATED)

JOBS_DEPS = jobs/Dockerfile

ARV_WEB_DEPS = arv-web/Dockerfile arv-web/apache2_foreground.sh arv-web/apache2_vhost

JAVA_BWA_SAMTOOLS_DEPS = java-bwa-samtools/Dockerfile

API_DEPS = api/* config.yml $(API_GENERATED)

SHELL_DEPS = shell/* config.yml $(SHELL_GENERATED)

COMPUTE_DEPS = compute/* config.yml $(COMPUTE_GENERATED)

DOC_DEPS = doc/Dockerfile $(DOC_GENERATED)

WORKBENCH_DEPS = workbench/Dockerfile \
                 config.yml \
                 $(WORKBENCH_GENERATED)

KEEP_DEPS = keep/Dockerfile config.yml $(KEEP_GENERATED)

KEEP_PROXY_DEPS = keepproxy/Dockerfile config.yml $(KEEP_PROXY_GENERATED)

SSO_DEPS = config.yml $(SSO_GENERATED)

BCBIO_NEXTGEN_DEPS = bcbio-nextgen/Dockerfile

BASE_GENERATED = base/generated/arvados.tar.gz

COMPUTE_GENERATED_IN   = compute/*.in
COMPUTE_GENERATED      = compute/generated/*

KEEP_GENERATED_IN      = keep/*.in
KEEP_GENERATED         = keep/generated/*

KEEP_PROXY_GENERATED_IN      = keepproxy/*.in
KEEP_PROXY_GENERATED         = keepproxy/generated/*

API_GENERATED_IN       = api/*.in
API_GENERATED          = api/generated/*

SHELL_GENERATED_IN     = shell/*.in
SHELL_GENERATED        = shell/generated/*

SLURM_GENERATED_IN     = slurm/*.in
SLURM_GENERATED        = slurm/generated/*

WORKBENCH_GENERATED_IN = workbench/*.in
WORKBENCH_GENERATED    = workbench/generated/*

SSO_GENERATED_IN       = sso/*.in
SSO_GENERATED          = sso/generated/*

DOC_GENERATED_IN       = doc/*.in
DOC_GENERATED          = doc/generated/*

KEEP_DEPS += keep/generated/bin/keepproxy
KEEP_DEPS += keep/generated/bin/keepstore
keep/generated/bin/%: $(wildcard build/services/%/*.go)
	mkdir -p keep/generated/src/git.curoverse.com
	ln -sfn ../../../../.. keep/generated/src/git.curoverse.com/arvados.git
	GOPATH=$(shell pwd)/keep/generated go get $(@:keep/generated/bin/%=git.curoverse.com/arvados.git/services/%)

KEEP_PROXY_DEPS += keepproxy/generated/bin/keepproxy
keepproxy/generated/bin/%: $(wildcard build/services/%/*.go)
	mkdir -p keepproxy/generated/src/git.curoverse.com
	ln -sfn ../../../../.. keepproxy/generated/src/git.curoverse.com/arvados.git
	GOPATH=$(shell pwd)/keepproxy/generated go get $(@:keepproxy/generated/bin/%=git.curoverse.com/arvados.git/services/%)

$(BUILD):
	mkdir -p build
	rsync -rlp --exclude=docker/ --exclude='**/log/*' --exclude='**/tmp/*' \
		--chmod=Da+rx,Fa+rX ../ build/
	find build/ -name \*.gem -delete
	cd build/services/fuse/ && python setup.py build
	cd build/sdk/python/ && python setup.py build
	cd build/sdk/cli && gem build arvados-cli.gemspec
	cd build/sdk/ruby && gem build arvados.gemspec
	touch build/.buildstamp

$(SLURM_GENERATED): $(BUILD)
	$(CONFIG_RB) slurm
	mkdir -p slurm/generated

$(BASE_GENERATED): $(BUILD)
	$(CONFIG_RB) base
	mkdir -p base/generated
	tar -czf base/generated/arvados.tar.gz -C build .

$(API_GENERATED): $(API_GENERATED_IN)
	$(CONFIG_RB) api

$(SHELL_GENERATED): $(SHELL_GENERATED_IN)
	$(CONFIG_RB) shell

$(WORKBENCH_GENERATED): $(WORKBENCH_GENERATED_IN)
	$(CONFIG_RB) workbench

$(COMPUTE_GENERATED): $(COMPUTE_GENERATED_IN)
	$(CONFIG_RB) compute

$(SSO_GENERATED): $(SSO_GENERATED_IN)
	$(CONFIG_RB) sso

$(DOC_GENERATED): $(DOC_GENERATED_IN)
	$(CONFIG_RB) doc

$(KEEP_GENERATED): $(KEEP_GENERATED_IN)
	$(CONFIG_RB) keep

$(KEEP_PROXY_GENERATED): $(KEEP_PROXY_GENERATED_IN)
	$(CONFIG_RB) keepproxy

DOCKER_BUILD = $(DOCKER) build --rm=true

# ============================================================
# The main Arvados servers: api, doc, workbench, compute

api-image: passenger-image $(BUILD) $(API_DEPS)
	@echo "Building api-image"
	mkdir -p api/generated
	tar -czf api/generated/api.tar.gz -C build/services api
	$(DOCKER_BUILD) -t arvados/api api
	date >api-image

shell-image: base-image $(BUILD) $(SHELL_DEPS)
	@echo "Building shell-image"
	mkdir -p shell/generated
	$(DOCKER_BUILD) -t arvados/shell shell
	date >shell-image

compute-image: slurm-image $(BUILD) $(COMPUTE_DEPS)
	@echo "Building compute-image"
	$(DOCKER_BUILD) -t arvados/compute compute
	date >compute-image

doc-image: base-image $(BUILD) $(DOC_DEPS)
	@echo "Building doc-image"
	mkdir -p doc/generated
	tar -czf doc/generated/doc.tar.gz -C build doc
	$(DOCKER_BUILD) -t arvados/doc doc
	date >doc-image

keep-image: debian-arvados-image $(BUILD) $(KEEP_DEPS)
	@echo "Building keep-image"
	$(DOCKER_BUILD) -t arvados/keep keep
	date >keep-image

keep-proxy-image: debian-arvados-image $(BUILD) $(KEEP_PROXY_DEPS)
	@echo "Building keep-proxy-image"
	$(DOCKER_BUILD) -t arvados/keepproxy keepproxy
	date >keep-proxy-image

jobs-image: debian-arvados-image $(BUILD) $(JOBS_DEPS)
	$(DOCKER_BUILD) -t arvados/jobs jobs
	date >jobs-image

java-bwa-samtools-image: jobs-image $(BUILD) $(JAVA_BWA_SAMTOOLS_DEPS)
	$(DOCKER_BUILD) -t arvados/jobs-java-bwa-samtools java-bwa-samtools
	date >java-bwa-samtools-image

bcbio-nextgen-image: $(BUILD) $(BASE_GENERATED) $(BCBIO_NEXTGEN_DEPS)
	rm -rf bcbio-nextgen/generated
	cp -r base/generated bcbio-nextgen
	$(DOCKER_BUILD) -t arvados/bcbio-nextgen bcbio-nextgen
	date >bcbio-nextgen-image

workbench-image: passenger-image $(BUILD) $(WORKBENCH_DEPS)
	@echo "Building workbench-image"
	mkdir -p workbench/generated
	tar -czf workbench/generated/workbench.tar.gz -C build/apps workbench
	$(DOCKER_BUILD) -t arvados/workbench workbench
	date >workbench-image

sso-image: passenger-image $(SSO_DEPS)
	@echo "Building sso-image"
	$(DOCKER_BUILD) -t arvados/sso sso
	date >sso-image

arv-web-image: passenger-image $(ARV_WEB_DEPS)
	$(DOCKER_BUILD) -t arvados/arv-web arv-web
	date >arv-web-image

# ============================================================
# The arvados/base image is the base Debian image plus packages
# that are dependencies for every Arvados service.

passenger-image: base-image
	@echo "Building passenger-image"
	$(DOCKER_BUILD) -t arvados/passenger passenger
	date >passenger-image

slurm-image: base-image $(SLURM_DEPS)
	@echo "Building slurm-image"
	$(DOCKER_BUILD) -t arvados/slurm slurm
	date >slurm-image

base-image: debian-arvados-image $(BASE_DEPS)
	@echo "Building base-image"
	$(DOCKER_BUILD) -t arvados/base base
	date >base-image

debian-arvados-image:
	@echo "Building debian-arvados-image"
	./mkimage-debootstrap.sh arvados/debian wheezy http://ftp.us.debian.org/debian/
	date >debian-arvados-image

skydns-image:
	@echo "Downloading skydns-image"
	$(DOCKER) pull crosbymichael/skydns
	date >skydns-image

skydock-image:
	@echo "Downloading skydock-image"
	$(DOCKER) pull crosbymichael/skydock
	date >skydock-image
