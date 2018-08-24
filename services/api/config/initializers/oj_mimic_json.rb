# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'oj'

Oj::Rails.set_encoder()
Oj::Rails.set_decoder()
Oj::Rails.optimize()
Oj::Rails.mimic_JSON()

