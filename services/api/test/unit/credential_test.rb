# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require "test_helper"

class CredentialTest < ActiveSupport::TestCase
  setup do
    @valid_attrs = {
      name: "My Credential",
      description: "Test credential",
      credential_class: "basic_auth",
      external_id: "user123",
      secret: "secret_value",
      expires_at: Time.now + 1.week
    }
  end

  test "valid credential with all required fields" do
    credential = Credential.new(@valid_attrs)
    assert credential.valid?
  end

  test "invalid without required fields" do
    [:name, :credential_class, :external_id, :secret, :expires_at].each do |field|
      attrs = @valid_attrs.dup
      attrs[field] = nil
      credential = Credential.new(attrs)
      refute credential.valid?, "#{field} should be required"
      assert_includes credential.errors.keys, field
    end
  end

  test "scopes defaults to empty array" do
    credential = Credential.new(@valid_attrs)
    assert_equal [], credential.scopes
  end

  test "logged_attributes excludes secret" do
    credential = Credential.create!(@valid_attrs.merge(owner_uuid: SecureRandom.uuid))
    logged_attrs = credential.logged_attributes
    refute logged_attrs.key?("secret")
  end

  test "ensure_owner_uuid_is_permitted sets owner to system_user for new record" do
    credential = Credential.new(@valid_attrs.merge(owner_uuid: SecureRandom.uuid))
    system_uuid = Credential.system_user_uuid
    assert credential.ensure_owner_uuid_is_permitted
    assert_equal system_uuid, credential.owner_uuid
  end

  test "ensure_owner_uuid_is_permitted raises if owner_uuid is not system user" do
    credential = Credential.create!(@valid_attrs.merge(owner_uuid: SecureRandom.uuid))
    credential.owner_uuid = "some_other_uuid"
    assert_raises RuntimeError do
      credential.ensure_owner_uuid_is_permitted
    end
  end
end
