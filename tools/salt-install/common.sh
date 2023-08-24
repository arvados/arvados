##########################################################
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

# This is generic logic used by provision.sh & installer.sh scripts

if [[ -s ${CONFIG_FILE} && -s ${CONFIG_FILE}.secrets ]]; then
  source ${CONFIG_FILE}.secrets
  source ${CONFIG_FILE}
else
  echo >&2 "You don't seem to have a config file with initial values."
  echo >&2 "Please create a '${CONFIG_FILE}' & '${CONFIG_FILE}.secrets' files as described in"
  echo >&2 "  * https://doc.arvados.org/install/salt-single-host.html#single_host, or"
  echo >&2 "  * https://doc.arvados.org/install/salt-multi-host.html#multi_host_multi_hostnames"
  exit 1
fi

USE_SSH_JUMPHOST=${USE_SSH_JUMPHOST:-}

# Comma-separated list of nodes. This is used to dynamically adjust
# salt pillars.
NODELIST=""
for node in "${!NODES[@]}"; do
  if [ -z "$NODELIST" ]; then
    NODELIST="$node"
  else
    NODELIST="$NODELIST,$node"
  fi
done

# The mapping of roles to nodes. This is used to dynamically adjust
# salt pillars.
for node in "${!NODES[@]}"; do
  roles="${NODES[$node]}"

  # Split the comma-separated roles into an array
  IFS=',' read -ra roles_array <<< "$roles"

  for role in "${roles_array[@]}"; do
    if [ -n "${ROLE2NODES[$role]:-}" ]; then
      ROLE2NODES["$role"]="${ROLE2NODES[$role]},$node"
    else
      ROLE2NODES["$role"]=$node
    fi
  done
done

# Auto-detects load-balancing mode
if [ -z "${ROLE2NODES['balancer']:-}" ]; then
  ENABLE_BALANCER="no"
else
  ENABLE_BALANCER="yes"
fi
