require 'test_helper'
load 'test/functional/arvados/v1/git_setup.rb'

class Arvados::V1::CommitsControllerTest < ActionController::TestCase
  fixtures :repositories, :users
  
  include GitSetup
  
  test "test_find_commit_range" do
    authorize_with :active

  # single
    a = Commit.find_commit_range(users(:active), nil, nil, '31ce37fe365b3dc204300a3e4c396ad333ed0556')
    assert_equal ['31ce37fe365b3dc204300a3e4c396ad333ed0556'], a

  #test "test_branch1" do
    a = Commit.find_commit_range(users(:active), nil, nil, 'master')
    assert_equal ['077ba2ad3ea24a929091a9e6ce545c93199b8e57'], a

  #test "test_branch2" do
    a = Commit.find_commit_range(users(:active), 'foo', nil, 'b1')
    assert_equal ['1de84a854e2b440dc53bf42f8548afa4c17da332'], a

  #test "test_branch3" do
    a = Commit.find_commit_range(users(:active), 'foo', nil, 'HEAD')
    assert_equal ['1de84a854e2b440dc53bf42f8548afa4c17da332'], a

  #test "test_single_revision_repo" do
    a = Commit.find_commit_range(users(:active), "foo", nil, '31ce37fe365b3dc204300a3e4c396ad333ed0556')
    assert_equal ['31ce37fe365b3dc204300a3e4c396ad333ed0556'], a
    a = Commit.find_commit_range(users(:active), "bar", nil, '31ce37fe365b3dc204300a3e4c396ad333ed0556')
    assert_equal nil, a

  #test "test_multi_revision" do
    a = Commit.find_commit_range(users(:active), nil, '31ce37fe365b3dc204300a3e4c396ad333ed0556', '077ba2ad3ea24a929091a9e6ce545c93199b8e57')
    assert_equal ['077ba2ad3ea24a929091a9e6ce545c93199b8e57', '4fe459abe02d9b365932b8f5dc419439ab4e2577', '31ce37fe365b3dc204300a3e4c396ad333ed0556'], a

  #test "test_tag" do
    a = Commit.find_commit_range(users(:active), nil, 'tag1', 'master')
    assert_equal ['077ba2ad3ea24a929091a9e6ce545c93199b8e57', '4fe459abe02d9b365932b8f5dc419439ab4e2577'], a
  end

end
