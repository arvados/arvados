# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Credential < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :description
    t.add :credential_class
    t.add :credential_id
    t.add :expires_at
  end

end
