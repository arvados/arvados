# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Credential < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  attribute :scopes, :jsonbArray, default: []

  after_create :add_credential_manage_link

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :description
    t.add :credential_class
    t.add :scopes
    t.add :external_id
    t.add :expires_at
  end

  def updated_at=(v)
      # no-op
  end

  def logged_attributes
    super.except('secret')
  end

  def self.full_text_searchable_columns
    super - ["credential_class", "external_id", "secret", "expires_at"]
  end

  def self.searchable_columns *args
    super - ["secret", "expires_at"]
  end

  def ensure_owner_uuid_is_permitted
    if new_record?
      @requested_manager_uuid = owner_uuid
      self.owner_uuid = system_user_uuid
      return true
    end

    if self.owner_uuid != system_user_uuid
      raise "Owner uuid for credential must be system user"
    end
  end

  def add_credential_manage_link
    if @requested_manager_uuid
      act_as_system_user do
       Link.create!(tail_uuid: @requested_manager_uuid,
                    head_uuid: self.uuid,
                    link_class: "permission",
                    name: "can_manage")
      end
    end
  end

end
