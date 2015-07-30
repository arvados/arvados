require 'test_helper'

class AuthorizedKeyTest < ActiveSupport::TestCase
  TEST_KEY = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCf5aTI55uyWr44TckP/ELUAyPsdnf5fTZDcSDN4qiMZYAL7TYV2ixwnbPObLObM0GmHSSFLV1KqsuFICUPgkyKoHbAH6XPgmtfOLU60VkGf1v5uxQ/kXCECRCJmPb3K9dIXGEw+1DXPdOV/xG7rJNvo4a9WK9iqqZr8p+VGKM6C017b8BDLk0tuEEjZ5jXcT/ka/hTScxWkKgF6auPOVQ79OA5+0VaYm4uQLzVUdgwVUPWQQecRrtnc08XYM1htpcLDIAbWfUNK7uE6XR3/OhtrJGf05FGbtGguPgi33F9W3Q3yw6saOK5Y3TfLbskgFaEdLgzqK/QSBRk2zBF49Tj test@localhost"

  test 'create and update key' do
    u1 = users(:active)
    act_as_user u1 do
      ak = AuthorizedKey.new(name: "foo", public_key: TEST_KEY, authorized_user_uuid: u1.uuid)
      assert ak.save, ak.errors.full_messages.to_s
      ak.name = "bar"
      assert ak.valid?, ak.errors.full_messages.to_s
      assert ak.save, ak.errors.full_messages.to_s
    end
  end

  test 'duplicate key not permitted' do
    u1 = users(:active)
    act_as_user u1 do
      ak = AuthorizedKey.new(name: "foo", public_key: TEST_KEY, authorized_user_uuid: u1.uuid)
      assert ak.save
    end
    u2 = users(:spectator)
    act_as_user u2 do
      ak2 = AuthorizedKey.new(name: "bar", public_key: TEST_KEY, authorized_user_uuid: u2.uuid)
      refute ak2.valid?
      refute ak2.save
      assert_match /already exists/, ak2.errors.full_messages.to_s
    end
  end

  test 'attach key to wrong user account' do
    act_as_user users(:active) do
      ak = AuthorizedKey.new(name: "foo", public_key: TEST_KEY)
      ak.authorized_user_uuid = users(:spectator).uuid
      refute ak.save
      ak.uuid = nil
      ak.authorized_user_uuid = users(:admin).uuid
      refute ak.save
      ak.uuid = nil
      ak.authorized_user_uuid = users(:active).uuid
      assert ak.save, ak.errors.full_messages.to_s
      ak.authorized_user_uuid = users(:admin).uuid
      refute ak.save
    end
  end
end
