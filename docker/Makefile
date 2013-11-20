all: api-image docserver-image workbench-image warehouse-image sso-image

clean:
	-rm *-image */generated/*

# ============================================================
# Dependencies for */generated files which are prerequisites
# for building docker images.

API_DEPS = api/Dockerfile $(API_GENERATED)

DOCSERVER_DEPS = docserver/Dockerfile docserver/apache2_vhost

WORKBENCH_DEPS = workbench/Dockerfile \
                 workbench/passenger.conf \
                 $(WORKBENCH_GENERATED)

WAREHOUSE_DEPS = warehouse/Dockerfile \
                 warehouse/supervisor.conf \
                 $(WAREHOUSE_GENERATED)

SSO_DEPS = sso/passenger.conf $(SSO_GENERATED)

API_GENERATED = \
        api/generated/apache2_vhost \
        api/generated/config_databases.sh \
        api/generated/database.yml \
        api/generated/omniauth.rb \
        api/generated/production.rb \
        api/generated/secret_token.rb

API_GENERATED_IN = \
        api/apache2_vhost.in \
        api/config_databases.sh.in \
        api/database.yml.in \
        api/omniauth.rb.in \
        api/production.rb.in \
        api/secret_token.rb.in

WORKBENCH_GENERATED = \
        workbench/generated/apache2_vhost \
        workbench/generated/production.rb \
        workbench/generated/secret_token.rb

WORKBENCH_GENERATED_IN = \
        workbench/apache2_vhost.in \
        workbench/production.rb.in \
        workbench/secret_token.rb.in

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

$(API_GENERATED): $(API_GENERATED_IN)
	./config.rb

$(WORKBENCH_GENERATED): $(WORKBENCH_GENERATED_IN)
	./config.rb

$(WAREHOUSE_GENERATED): $(WAREHOUSE_GENERATED_IN)
	./config.rb

$(SSO_GENERATED): $(SSO_GENERATED_IN)
	./config.rb

# ============================================================
# The main Arvados servers: api, docserver, workbench, warehouse

api-image: passenger-image $(API_DEPS)
	mkdir -p api/generated
	tar -c -z -f api/generated/api.tar.gz -C ../services api
	./docker_build -t arvados/api api
	echo -n "Built at $(date)" > api-image

docserver-image: base-image $(DOCSERVER_DEPS)
	mkdir -p docserver/generated
	tar -c -z -f docserver/generated/doc.tar.gz -C .. doc
	./docker_build -t arvados/docserver docserver
	echo -n "Built at $(date)" > docserver-image

workbench-image: passenger-image $(WORKBENCH_DEPS)
	mkdir -p workbench/generated
	tar -c -z -f workbench/generated/workbench.tar.gz -C ../apps workbench
	./docker_build -t arvados/workbench workbench
	echo -n "Built at $(date)" > workbench-image

warehouse-image: base-image $(WAREHOUSE_DEPS)
	./docker_build -t arvados/warehouse warehouse
	echo -n "Built at $(date)" > warehouse-image

sso-image: passenger-image $(SSO_DEPS)
	./docker_build -t arvados/sso sso
	echo -n "Built at $(date)" > sso-image

# ============================================================
# The arvados/base image is the base Debian image plus packages
# that are dependencies for every Arvados service.

passenger-image: base-image
	./docker_build -t arvados/passenger passenger
	echo -n "Built at $(date)" > passenger-image

base-image: debian-image
	./docker_build -t arvados/base base
	echo -n "Built at $(date)" > base-image

debian-image:
	./mkimage-debootstrap.sh arvados/debian wheezy http://debian.lcs.mit.edu/debian/
	echo -n "Built at $(date)" > debian-image

