# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require_relative "../../app/models/jsonb_type"

# JSONB backed Hash & Array types that default to their empty versions when
# reading NULL from the database, or get nil passed by parameter.
ActiveRecord::Type.register(:jsonbHash, JsonbType::Hash)
ActiveRecord::Type.register(:jsonbArray, JsonbType::Array)
