all: api docserver workbench warehouse

# ============================================================
# The main Arvados servers: api, docserver, workbench, warehouse

api: .built-api

.built-api: base $(API_DEPS)
	docker build -t arvados/api api && touch .built-api

docserver: .built-docserver

.built-docserver: base $(DOCSERVER_DEPS) 
	docker build -t arvados/docserver docserver && touch .built-docserver

workbench: .built-workbench

.built-workbench: base $(WORKBENCH_DEPS)
	docker build -t arvados/workbench workbench && touch .built-workbench

warehouse: .built-warehouse

.built-warehouse: base $(WAREHOUSE_DEPS)
	docker build -t arvados/warehouse warehouse && touch .built-warehouse

# ============================================================
# The arvados/base image is the base Debian image plus packages
# that are dependencies for every Arvados service.

base:	.built-base

.built-base:	debian
	docker build -t arvados/base base && touch .built-base

debian:		.built-debian

.built-debian:
	./mkimage-debootstrap.sh arvados/debian wheezy http://debian.lcs.mit.edu/debian/ \
	&& touch .built-debian

# ============================================================
# Dependencies for */generated files which are prerequisites
# for building docker images.

API_DEPS = api/generated/api.tar.gz $(API_GENERATED)

DOCSERVER_DEPS = docserver/generated/doc.tar.gz docserver/apache2_vhost

WORKBENCH_DEPS = workbench/generated/workbench.tar.gz \
                 workbench/passenger.conf \
                 $(WORKBENCH_GENERATED)

WAREHOUSE_DEPS = warehouse/generated/warehouse.tar.gz \
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

api/generated/api.tar.gz:
	tar -c -z -f api/generated/api.tar.gz -C ../services api

docserver/generated/doc.tar.gz:
	tar -c -z -f docserver/generated/doc.tar.gz -C .. doc

workbench/generated/workbench.tar.gz:
	tar -c -z -f workbench/generated/workbench.tar.gz -C ../apps workbench

warehouse/generated/warehouse.tar.gz:
	tar -c -z -f warehouse/generated/warehouse.tar.gz -C ../.. warehouse-apps
$(API_GENERATED): $(API_GENERATED_IN)
	./config.rb

$(WORKBENCH_GENERATED): $(WORKBENCH_GENERATED_IN)
	./config.rb

$(WAREHOUSE_GENERATED): $(WAREHOUSE_GENERATED_IN)
	./config.rb
