# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Credential < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  attribute :credential_scopes, :jsonbArray, default: []

  before_save :check_expires_at

  after_create :add_credential_manage_link

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :description
    t.add :credential_class
    t.add :credential_scopes
    t.add :credential_id
    t.add :expires_at
  end

  def updated_at=(v)
      # no-op
  end

  def logged_attributes
    super.except('credential_secret')
  end

  def self.full_text_searchable_columns
    super - ["credential_class", "credential_id", "credential_secret"]
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

  def check_expires_at
    if expires_at.nil?
      raise ArgumentError.new "expires_at cannot be nil"
    end
    if !new_record? && expires_at > expires_at_was && credential_secret == credential_secret_was
      raise ArgumentError.new "can only set expires_at further into the future when changing credential_secret"
    end
    if Time.now >= expires_at && !credential_secret.empty?
      if credential_secret == credential_secret_was
        raise ArgumentError.new "credential has expired, this credential can only be updated if credential_secret is updated"
      else
        raise ArgumentError.new "when updating credential_secret, must also set expires_at to a time in the future"
      end
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
