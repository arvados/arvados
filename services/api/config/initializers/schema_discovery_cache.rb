# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Delete the cached discovery document during startup. Otherwise we
# might still serve an old discovery document after updating the
# schema and restarting the server.

Rails.cache.delete 'arvados_v1_rest_discovery'
