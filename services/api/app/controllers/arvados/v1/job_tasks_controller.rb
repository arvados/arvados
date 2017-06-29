# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::JobTasksController < ApplicationController
  accept_attribute_as_json :parameters, Hash
end
