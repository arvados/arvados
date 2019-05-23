# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'config_validators'

include ConfigValidators

ConfigValidators::validate_wb2_url_config()