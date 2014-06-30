all: api-image doc-image workbench-image warehouse-image sso-image

# `make clean' removes the files generated in the build directory
# but does not remove any docker images generated in previous builds
clean:
	-rm -rf build
	-rm *-image */generated/*
	-@rmdir */generated

# `make realclean' will also remove the docker images and force
# subsequent makes to build the entire chain from the ground up
realclean: clean
	-[ -n "`$(DOCKER) ps -q`" ] && $(DOCKER) stop `$(DOCKER) ps -q`
	-$(DOCKER) rm `$(DOCKER) ps -a -q`
	-$(DOCKER) rmi `$(DOCKER) images -q`

# ============================================================
# Dependencies for */generated files which are prerequisites
# for building docker images.

CONFIG_RB = build_tools/config.rb

BUILD = build/.buildstamp

BASE_DEPS = base/Dockerfile $(BASE_GENERATED)

JOBS_DEPS = jobs/Dockerfile

API_DEPS = api/Dockerfile $(API_GENERATED)

DOC_DEPS = doc/Dockerfile doc/apache2_vhost

WORKBENCH_DEPS = workbench/Dockerfile \
                 workbench/passenger.conf \
                 $(WORKBENCH_GENERATED)

WAREHOUSE_DEPS = warehouse/Dockerfile \
                 warehouse/supervisor.conf \
                 $(WAREHOUSE_GENERATED)

SSO_DEPS = sso/passenger.conf $(SSO_GENERATED)

BASE_GENERATED = base/generated/arvados.tar.gz

API_GENERATED = \
        api/generated/apache2_vhost \
        api/generated/config_databases.sh \
        api/generated/database.yml \
        api/generated/omniauth.rb \
        api/generated/application.yml \
        api/generated/superuser_token

API_GENERATED_IN = \
        api/apache2_vhost.in \
        api/config_databases.sh.in \
        api/database.yml.in \
        api/omniauth.rb.in \
        api/application.yml.in \
        api/superuser_token.in

WORKBENCH_GENERATED = \
        workbench/generated/apache2_vhost \
        workbench/generated/application.yml

WORKBENCH_GENERATED_IN = \
        workbench/apache2_vhost.in \
        workbench/application.yml.in

WAREHOUSE_GENERATED = warehouse/generated/warehouse.conf

WAREHOUSE_GENERATED_IN = warehouse/warehouse.conf.in

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

$(BASE_GENERATED): config.yml $(BUILD)
	$(CONFIG_RB)
	mkdir -p base/generated
	tar -czf base/generated/arvados.tar.gz -C build .

$(API_GENERATED): config.yml $(API_GENERATED_IN)
	$(CONFIG_RB)

$(WORKBENCH_GENERATED): config.yml $(WORKBENCH_GENERATED_IN)
	$(CONFIG_RB)

$(WAREHOUSE_GENERATED): config.yml $(WAREHOUSE_GENERATED_IN)
	$(CONFIG_RB)

$(SSO_GENERATED): config.yml $(SSO_GENERATED_IN)
	$(CONFIG_RB)

# The docker build -q option suppresses verbose build output.
# Necessary to prevent failure on building warehouse; see
# https://github.com/dotcloud/docker/issues/3172
DOCKER_BUILD = $(DOCKER) build -q

# ============================================================
# The main Arvados servers: api, doc, workbench, warehouse

api-image: passenger-image $(BUILD) $(API_DEPS)
	mkdir -p api/generated
	tar -czf api/generated/api.tar.gz -C build/services api
	$(DOCKER_BUILD) -t arvados/api api
	date >api-image

doc-image: base-image $(BUILD) $(DOC_DEPS)
	mkdir -p doc/generated
	tar -czf doc/generated/doc.tar.gz -C build doc
	$(DOCKER_BUILD) -t arvados/doc doc
	date >doc-image

jobs-image: base-image $(BUILD) $(JOBS_DEPS)
	$(DOCKER_BUILD) -t arvados/jobs jobs
	date >jobs-image

workbench-image: passenger-image $(BUILD) $(WORKBENCH_DEPS)
	mkdir -p workbench/generated
	tar -czf workbench/generated/workbench.tar.gz -C build/apps workbench
	$(DOCKER_BUILD) -t arvados/workbench workbench
	date >workbench-image

warehouse-image: base-image $(WAREHOUSE_DEPS)
	$(DOCKER_BUILD) -t arvados/warehouse warehouse
	date >warehouse-image

sso-image: passenger-image $(SSO_DEPS)
	$(DOCKER_BUILD) -t arvados/sso sso
	date >sso-image

# ============================================================
# The arvados/base image is the base Debian image plus packages
# that are dependencies for every Arvados service.

passenger-image: base-image
	$(DOCKER_BUILD) -t arvados/passenger passenger
	date >passenger-image

base-image: debian-image $(BASE_DEPS)
	$(DOCKER_BUILD) -t arvados/base base
	date >base-image

debian-image:
	./mkimage-debootstrap.sh arvados/debian wheezy ftp://ftp.us.debian.org/debian/
	date >debian-image
