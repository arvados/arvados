# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class PipelineTemplate < ArvadosModel
  before_create :create_disabled
  before_update :update_disabled

  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :components, Hash

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :components
    t.add :description
  end

  def self.limit_index_columns_read
    ["components"]
  end

  def create_disabled
    raise "Disabled"
  end

  def update_disabled
    raise "Disabled"
  end
end
