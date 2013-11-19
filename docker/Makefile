all: api-image docserver-image workbench-image warehouse-image

# ============================================================
# The main Arvados servers: api, docserver, workbench, warehouse

api-image: base-image $(API_DEPS)
	tar -c -z -f api/generated/api.tar.gz -C ../services api
	docker build -t arvados/api api
	echo -n "Built at $(date)" > api-image

docserver-image: base-image $(DOCSERVER_DEPS) 
	tar -c -z -f docserver/generated/doc.tar.gz -C .. doc
	docker build -t arvados/docserver docserver
	echo -n "Built at $(date)" > docserver-image

workbench-image: base-image $(WORKBENCH_DEPS)
	tar -c -z -f workbench/generated/workbench.tar.gz -C ../apps workbench
	docker build -t arvados/workbench workbench
	echo -n "Built at $(date)" > workbench-image

warehouse-image: base-image $(WAREHOUSE_DEPS)
	docker build -t arvados/warehouse warehouse
	echo -n "Built at $(date)" > warehouse-image

# ============================================================
# The arvados/base image is the base Debian image plus packages
# that are dependencies for every Arvados service.

base-image: debian-image
	docker build -t arvados/base base
	echo -n "Built at $(date)" > base-image

debian-image:
	./mkimage-debootstrap.sh arvados/debian wheezy http://debian.lcs.mit.edu/debian/
	echo -n "Built at $(date)" > debian-image

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
        workbench/generated/apache_vhost \
        workbench/generated/production.rb \
        workbench/generated/secret_token.rb

WORKBENCH_GENERATED_IN = \
        workbench/apache_vhost.in \
        workbench/production.rb.in \
        workbench/secret_token.rb.in

WAREHOUSE_GENERATED = warehouse/generated/warehouse.conf

WAREHOUSE_GENERATED_IN = warehouse/warehouse.conf.in

$(API_GENERATED): $(API_GENERATED_IN)
	./config.rb

$(WORKBENCH_GENERATED): $(WORKBENCH_GENERATED_IN)
	./config.rb

$(WAREHOUSE_GENERATED): $(WAREHOUSE_GENERATED_IN)
	./config.rb
