# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Load the rails application
require_relative 'application'
require 'josh_id'
require_relative 'arvados_config'

# Initialize the rails application
Rails.application.initialize!
