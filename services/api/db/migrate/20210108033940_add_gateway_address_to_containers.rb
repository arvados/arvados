# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddGatewayAddressToContainers < ActiveRecord::Migration[5.2]
  def change
    add_column :containers, :gateway_address, :string
  end
end
