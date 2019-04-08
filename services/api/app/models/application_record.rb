# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ApplicationRecord < ActiveRecord::Base
  self.abstract_class = true
end