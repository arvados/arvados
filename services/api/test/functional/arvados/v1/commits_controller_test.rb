require 'test_helper'
require 'helpers/git_test_helper'

# NOTE: calling Commit.find_commit_range(user, nil, nil, 'rev') will produce
# an error message "fatal: bad object 'rev'" on stderr if 'rev' does not exist
# in a given repository.  Many of these tests report such errors; their presence
# does not represent a fatal condition.
#
# TODO(twp): consider better error handling of these messages, or
# decide to abandon it.

class Arvados::V1::CommitsControllerTest < ActionController::TestCase
  fixtures :repositories, :users

  # See git_setup.rb for the commit log for test.git.tar
  include GitTestHelper

  test "test_find_commit_range" do
    authorize_with :active

  # single
    a = Commit.find_commit_range(users(:active), nil, nil, '31ce37fe365b3dc204300a3e4c396ad333ed0556', nil)
    assert_equal ['31ce37fe365b3dc204300a3e4c396ad333ed0556'], a

  #test "test_branch1" do
    # complains "fatal: bad object 077ba2ad3ea24a929091a9e6ce545c93199b8e57"
    a = Commit.find_commit_range(users(:active), nil, nil, 'master', nil)
    assert_includes(a, 'f35f99b7d32bac257f5989df02b9f12ee1a9b0d6')
    assert_includes(a, '077ba2ad3ea24a929091a9e6ce545c93199b8e57')

  #test "test_branch2" do
    a = Commit.find_commit_range(users(:active), 'active/foo', nil, 'b1', nil)
    assert_equal ['1de84a854e2b440dc53bf42f8548afa4c17da332'], a

  #test "test_branch3" do
    a = Commit.find_commit_range(users(:active), 'active/foo', nil, 'HEAD', nil)
    assert_equal ['1de84a854e2b440dc53bf42f8548afa4c17da332'], a

  #test "test_single_revision_repo" do
    a = Commit.find_commit_range(users(:active), "active/foo", nil, '31ce37fe365b3dc204300a3e4c396ad333ed0556', nil)
    assert_equal ['31ce37fe365b3dc204300a3e4c396ad333ed0556'], a
    a = Commit.find_commit_range(users(:active), "arvados", nil, '31ce37fe365b3dc204300a3e4c396ad333ed0556', nil)
    assert_equal nil, a

  #test "test_multi_revision" do
    # complains "fatal: bad object 077ba2ad3ea24a929091a9e6ce545c93199b8e57"
    a = Commit.find_commit_range(users(:active), nil, '31ce37fe365b3dc204300a3e4c396ad333ed0556', '077ba2ad3ea24a929091a9e6ce545c93199b8e57', nil)
    assert_equal ['077ba2ad3ea24a929091a9e6ce545c93199b8e57', '4fe459abe02d9b365932b8f5dc419439ab4e2577', '31ce37fe365b3dc204300a3e4c396ad333ed0556'], a

  #test "test_tag" do
    # complains "fatal: ambiguous argument 'tag1': unknown revision or path
    # not in the working tree."
    a = Commit.find_commit_range(users(:active), nil, 'tag1', 'master', nil)
    assert_equal ['077ba2ad3ea24a929091a9e6ce545c93199b8e57', '4fe459abe02d9b365932b8f5dc419439ab4e2577'], a

  #test "test_multi_revision_exclude" do
    a = Commit.find_commit_range(users(:active), nil, '31ce37fe365b3dc204300a3e4c396ad333ed0556', '077ba2ad3ea24a929091a9e6ce545c93199b8e57', ['4fe459abe02d9b365932b8f5dc419439ab4e2577'])
    assert_equal ['077ba2ad3ea24a929091a9e6ce545c93199b8e57', '31ce37fe365b3dc204300a3e4c396ad333ed0556'], a

  #test "test_multi_revision_tagged_exclude" do
    # complains "fatal: bad object 077ba2ad3ea24a929091a9e6ce545c93199b8e57"
    a = Commit.find_commit_range(users(:active), nil, '31ce37fe365b3dc204300a3e4c396ad333ed0556', '077ba2ad3ea24a929091a9e6ce545c93199b8e57', ['tag1'])
    assert_equal ['077ba2ad3ea24a929091a9e6ce545c93199b8e57', '31ce37fe365b3dc204300a3e4c396ad333ed0556'], a

    Dir.mktmpdir do |touchdir|
      # invalid input to maximum
      a = Commit.find_commit_range(users(:active), nil, nil, "31ce37fe365b3dc204300a3e4c396ad333ed0556 ; touch #{touchdir}/uh_oh", nil)
      assert !File.exists?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'maximum' parameter of find_commit_range is exploitable"
      assert_equal nil, a

      # invalid input to maximum
      a = Commit.find_commit_range(users(:active), nil, nil, "$(uname>#{touchdir}/uh_oh)", nil)
      assert !File.exists?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'maximum' parameter of find_commit_range is exploitable"
      assert_equal nil, a

      # invalid input to minimum
      a = Commit.find_commit_range(users(:active), nil, "31ce37fe365b3dc204300a3e4c396ad333ed0556 ; touch #{touchdir}/uh_oh", "31ce37fe365b3dc204300a3e4c396ad333ed0556", nil)
      assert !File.exists?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'minimum' parameter of find_commit_range is exploitable"
      assert_equal nil, a

      # invalid input to minimum
      a = Commit.find_commit_range(users(:active), nil, "$(uname>#{touchdir}/uh_oh)", "31ce37fe365b3dc204300a3e4c396ad333ed0556", nil)
      assert !File.exists?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'minimum' parameter of find_commit_range is exploitable"
      assert_equal nil, a

      # invalid input to 'excludes'
      # complains "fatal: bad object 077ba2ad3ea24a929091a9e6ce545c93199b8e57"
      a = Commit.find_commit_range(users(:active), nil, "31ce37fe365b3dc204300a3e4c396ad333ed0556", "077ba2ad3ea24a929091a9e6ce545c93199b8e57", ["4fe459abe02d9b365932b8f5dc419439ab4e2577 ; touch #{touchdir}/uh_oh"])
      assert !File.exists?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'excludes' parameter of find_commit_range is exploitable"
      assert_equal nil, a

      # invalid input to 'excludes'
      # complains "fatal: bad object 077ba2ad3ea24a929091a9e6ce545c93199b8e57"
      a = Commit.find_commit_range(users(:active), nil, "31ce37fe365b3dc204300a3e4c396ad333ed0556", "077ba2ad3ea24a929091a9e6ce545c93199b8e57", ["$(uname>#{touchdir}/uh_oh)"])
      assert !File.exists?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'excludes' parameter of find_commit_range is exploitable"
      assert_equal nil, a

    end

  end

end
