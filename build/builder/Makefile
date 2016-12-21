arvados/builder:
	cat ../../tools/arvbox/lib/arvbox/docker/Dockerfile.base | perl -ne 'print unless /VOLUME|var.lib.gems/; exit if /ARG arvados_version/;' > Dockerfile.generated
	cat Dockerfile.append >>Dockerfile.generated
	docker build --tag=arvados/builder --file=Dockerfile.generated .
