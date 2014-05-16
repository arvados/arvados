require 'test_helper'

class LinkTest < ActiveSupport::TestCase
  fixtures :all

  setup do
    set_user_from_auth :admin_trustedclient
  end

  test 'name links with the same tail_uuid must be unique' do
    a = Link.create!(tail_uuid: groups(:afolder).uuid,
                     head_uuid: specimens(:owned_by_active_user).uuid,
                     link_class: 'name',
                     name: 'foo')
    assert a.valid?, a.errors.to_s
    assert_raises ActiveRecord::RecordNotUnique do
      b = Link.create!(tail_uuid: groups(:afolder).uuid,
                       head_uuid: specimens(:owned_by_active_user).uuid,
                       link_class: 'name',
                       name: 'foo')
    end
  end

  test 'name links with different tail_uuid need not be unique' do
    a = Link.create!(tail_uuid: groups(:afolder).uuid,
                     head_uuid: specimens(:owned_by_active_user).uuid,
                     link_class: 'name',
                     name: 'foo')
    assert a.valid?, a.errors.to_s
    b = Link.create!(tail_uuid: groups(:asubfolder).uuid,
                     head_uuid: specimens(:owned_by_active_user).uuid,
                     link_class: 'name',
                     name: 'foo')
    assert b.valid?, b.errors.to_s
    assert_not_equal(a.uuid, b.uuid,
                     "created two links and got the same uuid back.")
  end

  [nil, '', false].each do |name|
    test "name links cannot be renamed to name=#{name.inspect}" do
      a = Link.create!(tail_uuid: groups(:afolder).uuid,
                       head_uuid: specimens(:owned_by_active_user).uuid,
                       link_class: 'name',
                       name: 'temp')
      a.name = name
      assert a.invalid?, "invalid name was accepted as valid?"
    end

    test "name links cannot be created with name=#{name.inspect}" do
      a = Link.create(tail_uuid: groups(:afolder).uuid,
                      head_uuid: specimens(:owned_by_active_user).uuid,
                      link_class: 'name',
                      name: name)
      if a.name and !a.name.empty?
        assert a.valid?, "name automatically assigned, but record not valid?"
      else
        assert a.invalid?, "invalid name was accepted as valid?"
      end
    end
  end

  test "cannot delete an object referenced by links" do
    ob = Specimen.create
    link = Link.create(tail_uuid: users(:active).uuid,
                       head_uuid: ob.uuid,
                       link_class: 'test',
                       name: 'test')
    assert_raises(ActiveRecord::DeleteRestrictionError,
                  "should not delete #{ob.uuid} with link #{link.uuid}") do
      ob.destroy
    end
  end

  test "assign sequential generic name links" do
    group = Group.create!(group_class: 'folder')
    ob = Specimen.create!
    25.times do |n|
      link = Link.create!(link_class: 'name',
                          tail_uuid: group.uuid, head_uuid: ob.uuid)
      expect_name = 'New specimen' + (n==0 ? "" : " (#{n})")
      assert_equal expect_name, link.name, "Expected sequential generic names"
    end
  end

  test "assign sequential generic name links for a two-word model" do
    group = Group.create!(group_class: 'folder')
    ob = VirtualMachine.create!
    5.times do |n|
      link = Link.create!(link_class: 'name',
                          tail_uuid: group.uuid, head_uuid: ob.uuid)
      expect_name = 'New virtual machine' + (n==0 ? "" : " (#{n})")
      assert_equal expect_name, link.name, "Expected sequential generic names"
    end
  end

  test "cannot assign sequential generic name links for a bogus uuid type" do
    group = Group.create!(group_class: 'folder')
    link = Link.create(link_class: 'name',
                       tail_uuid: group.uuid,
                       head_uuid: 'zzzzz-abcde-123451234512345')
    assert link.invalid?, "gave a bogus uuid, got automatic name #{link.name}"
  end
end
