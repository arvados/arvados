# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Credential < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  # Validation regexes for scopes, keyed by credential_class.
  CRED_CLASS_SCOPES_VALIDATION_REGEX = {
    "arv:aws_access_key" => [
      %r{\As3://(\*|[a-z0-9][\-.a-z0-9]{1,61}[a-z0-9])\z}
    ],
  }.freeze

  validates :name, :credential_class, :external_id, :secret, :expires_at, presence: true
  validates :name, format: { without: /\A[ \t]*\z/ }
  validates :scopes, array_of_strings: true

  attribute :scopes, :jsonbArray, default: []

  validate :validate_credential_class_and_scopes

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
    super - ["secret"]
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

  private

  def validate_credential_class_and_scopes
    return unless credential_class.present?
    return unless credential_class.start_with?("arv:")

    if CRED_CLASS_SCOPES_VALIDATION_REGEX.key?(credential_class)
      scopes_are_valid_for_supported_credential_class
    else
      errors.add(:credential_class, "credential_class #{credential_class} is not implemented")
    end
  end

  def scopes_are_valid_for_supported_credential_class
    return if scopes.blank?

    patterns = CRED_CLASS_SCOPES_VALIDATION_REGEX[credential_class]

    invalid = scopes.reject do |scope|
      patterns.any? { |re| re.match?(scope) }
    end
    
    if invalid.any?
      errors.add(:scopes, "Credential class #{credential_class} does not allow scopes: #{invalid.join(', ')}")
    end
  end
end
