# This is the 'shell hack'. Call make with DUMP=1 to see the effect.
ifdef DUMP
OLD_SHELL := $(SHELL)
SHELL = $(warning [$@])$(OLD_SHELL) -x
endif

all: skydns-image skydock-image api-image compute-image doc-image workbench-image keep-image sso-image shell-image

IMAGE_FILES := $(shell ls *-image 2>/dev/null |grep -v debian-arvados-image)
GENERATED_FILES := $(shell ls */generated/* 2>/dev/null)
GENERATED_DIRS := $(shell ls */generated 2>/dev/null)

# `make clean' removes the files generated in the build directory
# but does not remove any docker images generated in previous builds
clean:
	@echo "make clean"
	-@rm -rf build
	+@[ "$(IMAGE_FILES)$(GENERATED_FILES)" = "" ] || rm $(IMAGE_FILES) $(GENERATED_FILES) 2>/dev/null
	+@[ "$(GENERATED_DIRS)" = "" ] || rmdir */generated 2>/dev/null

DEBIAN_IMAGE := $(shell $(DOCKER) images -q arvados/debian |head -n1)

REALCLEAN_CONTAINERS := $(shell $(DOCKER) ps -a |grep -e arvados -e api_server -e keep_server -e doc_server -e workbench_server |cut -f 1 -d' ')
REALCLEAN_IMAGES := $(shell $(DOCKER) images -q arvados/* |grep -v $(DEBIAN_IMAGE) 2>/dev/null)
DEEPCLEAN_IMAGES := $(shell $(DOCKER) images -q arvados/*)
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

BASE_DEPS = base/Dockerfile $(BASE_GENERATED)

SLURM_DEPS = slurm/Dockerfile $(SLURM_GENERATED)

JOBS_DEPS = jobs/Dockerfile

JAVA_BWA_SAMTOOLS_DEPS = java-bwa-samtools/Dockerfile

API_DEPS = api/* $(API_GENERATED)

SHELL_DEPS = shell/* $(SHELL_GENERATED)

COMPUTE_DEPS = compute/* $(COMPUTE_GENERATED)

DOC_DEPS = doc/Dockerfile doc/apache2_vhost

WORKBENCH_DEPS = workbench/Dockerfile \
                 $(WORKBENCH_GENERATED)

KEEP_DEPS = keep/Dockerfile $(KEEP_GENERATED)

SSO_DEPS = $(SSO_GENERATED)

BCBIO_NEXTGEN_DEPS = bcbio-nextgen/Dockerfile

BASE_GENERATED = base/generated/arvados.tar.gz

SLURM_GENERATED = slurm/generated/*

COMPUTE_GENERATED = compute/generated/setup.sh

COMPUTE_GENERATED_IN = compute/setup.sh.in

KEEP_GENERATED_IN := $(shell ls keep/*.in)

KEEP_GENERATED := $(shell ls -1 $(KEEP_GENERATED_IN) | perl -pe 's:^keep:keep/generated:')

API_GENERATED = \
        api/generated/arvados-clients.yml \
        api/generated/apache2_vhost \
        api/generated/config_databases.sh \
        api/generated/database.yml \
        api/generated/omniauth.rb \
        api/generated/application.yml \
        api/generated/setup.sh \
        api/generated/setup-gitolite.sh \
        api/generated/slurm.conf \
        api/generated/superuser_token

API_GENERATED_IN = \
        api/arvados-clients.yml.in \
        api/apache2_vhost.in \
        api/config_databases.sh.in \
        api/database.yml.in \
        api/omniauth.rb.in \
        api/application.yml.in \
        api/setup.sh.in \
        api/setup-gitolite.sh.in \
        api/slurm.conf.in \
        api/superuser_token.in

SHELL_GENERATED = \
        shell/generated/setup.sh \
        shell/generated/superuser_token

SHELL_GENERATED_IN = \
        shell/setup.sh.in \
        shell/superuser_token.in

SLURM_GENERATED = \
        slurm/generated/slurm.conf

SLURM_GENERATED_IN = \
        slurm/slurm.conf.in

WORKBENCH_GENERATED = \
        workbench/generated/apache2_vhost \
        workbench/generated/application.yml

WORKBENCH_GENERATED_IN = \
        workbench/apache2_vhost.in \
        workbench/application.yml.in

SSO_GENERATED = \
        sso/generated/apache2_vhost \
        sso/generated/seeds.rb \
        sso/generated/secret_token.rb

SSO_GENERATED_IN = \
        sso/apache2_vhost.in \
        sso/seeds.rb.in \
        sso/secret_token.rb.in

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

$(SLURM_GENERATED): config.yml $(BUILD)
	$(CONFIG_RB)
	mkdir -p slurm/generated

$(BASE_GENERATED): config.yml $(BUILD)
	$(CONFIG_RB)
	mkdir -p base/generated
	tar -czf base/generated/arvados.tar.gz -C build .

$(API_GENERATED): config.yml $(API_GENERATED_IN)
	$(CONFIG_RB)

$(SHELL_GENERATED): config.yml $(SHELL_GENERATED_IN)
	$(CONFIG_RB)

$(WORKBENCH_GENERATED): config.yml $(WORKBENCH_GENERATED_IN)
	$(CONFIG_RB)

$(COMPUTE_GENERATED): config.yml $(COMPUTE_GENERATED_IN)
	$(CONFIG_RB)

$(WAREHOUSE_GENERATED): config.yml $(WAREHOUSE_GENERATED_IN)
	$(CONFIG_RB)

$(SSO_GENERATED): config.yml $(SSO_GENERATED_IN)
	$(CONFIG_RB)

$(KEEP_GENERATED): config.yml $(KEEP_GENERATED_IN)
	$(CONFIG_RB)

# The docker build -q option suppresses verbose build output.
# Necessary to prevent failure on building warehouse; see
# https://github.com/dotcloud/docker/issues/3172
DOCKER_BUILD = $(DOCKER) build -q --rm=true

# ============================================================
# The main Arvados servers: api, doc, workbench, warehouse

api-image: passenger-image $(BUILD) $(API_DEPS)
	@echo "Building api-image"
	mkdir -p api/generated
	tar -czf api/generated/api.tar.gz -C build/services api
	chmod 755 api/generated/setup.sh
	chmod 755 api/generated/setup-gitolite.sh
	$(DOCKER_BUILD) -t arvados/api api
	date >api-image

shell-image: base-image $(BUILD) $(SHELL_DEPS)
	@echo "Building shell-image"
	mkdir -p shell/generated
	chmod 755 shell/generated/setup.sh
	$(DOCKER_BUILD) -t arvados/shell shell
	date >shell-image

compute-image: slurm-image $(BUILD) $(COMPUTE_DEPS)
	@echo "Building compute-image"
	chmod 755 compute/generated/setup.sh
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

jobs-image: base-image $(BUILD) $(JOBS_DEPS)
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

warehouse-image: base-image $(WAREHOUSE_DEPS)
	$(DOCKER_BUILD) -t arvados/warehouse warehouse
	date >warehouse-image

sso-image: passenger-image $(SSO_DEPS)
	@echo "Building sso-image"
	$(DOCKER_BUILD) -t arvados/sso sso
	date >sso-image

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
	./mkimage-debootstrap.sh arvados/debian wheezy ftp://ftp.us.debian.org/debian/
	date >debian-arvados-image

skydns-image:
	@echo "Downloading skydns-image"
	$(DOCKER) pull crosbymichael/skydns
	date >skydns-image

skydock-image:
	@echo "Downloading skydock-image"
	$(DOCKER) pull crosbymichael/skydock
	date >skydock-image
