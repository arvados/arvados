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

  test "credential requires required fields" do
    [:name, :credential_class, :external_id, :secret, :expires_at].each do |field|
      attrs = @valid_attrs.dup
      good_credential = Credential.new(attrs)
      assert good_credential.valid?
      attrs[field] = nil
      bad_credential = Credential.new(attrs)
      assert_not bad_credential.valid?
    end
  end

  test "credential scopes defaults to empty array" do
    credential = Credential.new(@valid_attrs)
    assert_equal [], credential.scopes
  end

  test "credential logged_attributes excludes secret" do
    credential = nil
    act_as_system_user do
      credential = Credential.create!(@valid_attrs.merge(owner_uuid: system_user_uuid))
    end
    logged_attrs = credential.logged_attributes
    # required fields should always be logged
    assert_includes logged_attrs, "name"
    assert_includes logged_attrs, "credential_class"
    assert_includes logged_attrs, "external_id"
    assert_includes logged_attrs, "expires_at"

    # secret should never be logged
    refute logged_attrs.key?("secret")
  end

  test "credential ensure_owner_uuid_is_permitted sets owner to system_user for new record" do
    credential = Credential.new(@valid_attrs.merge(owner_uuid: SecureRandom.uuid))
    system_uuid = Credential.system_user_uuid
    assert credential.ensure_owner_uuid_is_permitted
    assert_equal system_uuid, credential.owner_uuid
  end

  test "credential ensure_owner_uuid_is_permitted raises if owner_uuid is not system user" do
    credential = nil
    act_as_system_user do
      credential = Credential.create!(@valid_attrs.merge(owner_uuid: system_user_uuid))
    end
    assert_nothing_raised do
      credential.ensure_owner_uuid_is_permitted
    end
    credential.owner_uuid = "some_other_uuid"
    assert_raises RuntimeError do
      credential.ensure_owner_uuid_is_permitted
    end
  end
end
