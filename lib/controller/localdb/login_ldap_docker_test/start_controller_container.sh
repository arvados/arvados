#!/bin/bash
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e
set -u
set -o pipefail

net_name="$1"; shift
tmpdir="$1"; shift
selfdir="$(readlink -e "$(dirname "$0")")"

docker run --detach --rm \
       --cidfile="$tmpdir/controller.cid" \
       --entrypoint=/run.sh \
       --network="$net_name" \
       -v "${tmpdir}/arvados.yml":/etc/arvados/config.yml:ro \
       -v "${tmpdir}/arvados-server":/bin/arvados-server:ro \
       -v "$(readlink -e ../../..)":/arvados:ro \
       -v "${selfdir}/run_controller.sh":/run.sh:ro \
       "$@" "$(cat "$tmpdir/controller_image")"

cont_addr="$(xargs -a "$tmpdir/controller.cid" docker inspect --format "{{(index .NetworkSettings.Networks \"${net_name}\").IPAddress}}")"
cont_url="http://${cont_addr}/arvados/v1/config"
for tries in $(seq 19 -1 0); do
    if curl -fsL "$cont_url" >/dev/null; then
        # Write the container address for the Go test code to record.
        # We had to get it here anyway so we might as well pass it up.
        echo "$cont_addr"
        exit
    elif [[ "$tries" != 0 ]]; then
        sleep 1
    fi
done

echo "error: controller did not come up" >&2
exit 7  # EXIT_NOTRUNNING
