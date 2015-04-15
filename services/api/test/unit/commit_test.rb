require 'test_helper'
require 'helpers/git_test_helper'

# NOTE: calling Commit.find_commit_range(nil, nil, 'rev')
# produces an error message "fatal: bad object 'rev'" on stderr if
# 'rev' does not exist in a given repository.  Many of these tests
# report such errors; their presence does not represent a fatal
# condition.

class CommitTest < ActiveSupport::TestCase
  # See git_setup.rb for the commit log for test.git.tar
  include GitTestHelper

  setup do
    authorize_with :active
  end

  test 'find_commit_range does not bypass permissions' do
    authorize_with :inactive
    assert_raises ArgumentError do
      c = Commit.find_commit_range 'foo', nil, 'master', []
    end
  end

  [
   'https://github.com/curoverse/arvados.git',
   'http://github.com/curoverse/arvados.git',
   'git://github.com/curoverse/arvados.git',
  ].each do |url|
    test "find_commit_range uses fetch_remote_repository to get #{url}" do
      fake_gitdir = repositories(:foo).server_path
      Commit.expects(:cache_dir_for).once.with(url).returns fake_gitdir
      Commit.expects(:fetch_remote_repository).once.with(fake_gitdir, url).returns true
      c = Commit.find_commit_range url, nil, 'master', []
      refute_empty c
    end
  end

  [
   'bogus/repo',
   '/bogus/repo',
   '/not/allowed/.git',
   'file:///not/allowed.git',
   'git.curoverse.com/arvados.git',
   'github.com/curoverse/arvados.git',
  ].each do |url|
    test "find_commit_range skips fetch_remote_repository for #{url}" do
      Commit.expects(:fetch_remote_repository).never
      assert_raises ArgumentError do
        Commit.find_commit_range url, nil, 'master', []
      end
    end
  end

  test 'fetch_remote_repository does not leak commits across repositories' do
    url = "http://localhost:1/fake/fake.git"
    fetch_remote_from_local_repo url, :foo
    c = Commit.find_commit_range url, nil, 'master', []
    assert_equal ['077ba2ad3ea24a929091a9e6ce545c93199b8e57'], c

    url = "http://localhost:2/fake/fake.git"
    fetch_remote_from_local_repo url, 'file://' + File.expand_path('../../.git', Rails.root)
    c = Commit.find_commit_range url, nil, '077ba2ad3ea24a929091a9e6ce545c93199b8e57', []
    assert_equal [], c
  end

  test 'tag_in_internal_repository creates and updates tags in internal.git' do
    authorize_with :active
    gitint = "git --git-dir #{Rails.configuration.git_internal_dir}"
    IO.read("|#{gitint} tag -d testtag 2>/dev/null") # "no such tag", fine
    assert_match /^fatal: /, IO.read("|#{gitint} show testtag 2>&1")
    refute $?.success?
    Commit.tag_in_internal_repository 'active/foo', '31ce37fe365b3dc204300a3e4c396ad333ed0556', 'testtag'
    assert_match /^commit 31ce37f/, IO.read("|#{gitint} show testtag")
    assert $?.success?
  end

  test "find_commit_range laundry list" do
    authorize_with :active

    # single
    a = Commit.find_commit_range('active/foo', nil, '31ce37fe365b3dc204300a3e4c396ad333ed0556', nil)
    assert_equal ['31ce37fe365b3dc204300a3e4c396ad333ed0556'], a

    #test "test_branch1" do
    a = Commit.find_commit_range('active/foo', nil, 'master', nil)
    assert_includes(a, '077ba2ad3ea24a929091a9e6ce545c93199b8e57')

    #test "test_branch2" do
    a = Commit.find_commit_range('active/foo', nil, 'b1', nil)
    assert_equal ['1de84a854e2b440dc53bf42f8548afa4c17da332'], a

    #test "test_branch3" do
    a = Commit.find_commit_range('active/foo', nil, 'HEAD', nil)
    assert_equal ['1de84a854e2b440dc53bf42f8548afa4c17da332'], a

    #test "test_single_revision_repo" do
    a = Commit.find_commit_range('active/foo', nil, '31ce37fe365b3dc204300a3e4c396ad333ed0556', nil)
    assert_equal ['31ce37fe365b3dc204300a3e4c396ad333ed0556'], a
    a = Commit.find_commit_range('arvados', nil, '31ce37fe365b3dc204300a3e4c396ad333ed0556', nil)
    assert_equal [], a

    #test "test_multi_revision" do
    # complains "fatal: bad object 077ba2ad3ea24a929091a9e6ce545c93199b8e57"
    a = Commit.find_commit_range('active/foo', '31ce37fe365b3dc204300a3e4c396ad333ed0556', '077ba2ad3ea24a929091a9e6ce545c93199b8e57', nil)
    assert_equal ['077ba2ad3ea24a929091a9e6ce545c93199b8e57', '4fe459abe02d9b365932b8f5dc419439ab4e2577', '31ce37fe365b3dc204300a3e4c396ad333ed0556'], a

    #test "test_tag" do
    # complains "fatal: ambiguous argument 'tag1': unknown revision or path
    # not in the working tree."
    a = Commit.find_commit_range('active/foo', 'tag1', 'master', nil)
    assert_equal ['077ba2ad3ea24a929091a9e6ce545c93199b8e57', '4fe459abe02d9b365932b8f5dc419439ab4e2577'], a

    #test "test_multi_revision_exclude" do
    a = Commit.find_commit_range('active/foo', '31ce37fe365b3dc204300a3e4c396ad333ed0556', '077ba2ad3ea24a929091a9e6ce545c93199b8e57', ['4fe459abe02d9b365932b8f5dc419439ab4e2577'])
    assert_equal ['077ba2ad3ea24a929091a9e6ce545c93199b8e57', '31ce37fe365b3dc204300a3e4c396ad333ed0556'], a

    #test "test_multi_revision_tagged_exclude" do
    # complains "fatal: bad object 077ba2ad3ea24a929091a9e6ce545c93199b8e57"
    a = Commit.find_commit_range('active/foo', '31ce37fe365b3dc204300a3e4c396ad333ed0556', '077ba2ad3ea24a929091a9e6ce545c93199b8e57', ['tag1'])
    assert_equal ['077ba2ad3ea24a929091a9e6ce545c93199b8e57', '31ce37fe365b3dc204300a3e4c396ad333ed0556'], a

    Dir.mktmpdir do |touchdir|
      # invalid input to maximum
      a = Commit.find_commit_range('active/foo', nil, "31ce37fe365b3dc204300a3e4c396ad333ed0556 ; touch #{touchdir}/uh_oh", nil)
      assert !File.exists?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'maximum' parameter of find_commit_range is exploitable"
      assert_equal [], a

      # invalid input to maximum
      a = Commit.find_commit_range('active/foo', nil, "$(uname>#{touchdir}/uh_oh)", nil)
      assert !File.exists?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'maximum' parameter of find_commit_range is exploitable"
      assert_equal [], a

      # invalid input to minimum
      a = Commit.find_commit_range('active/foo', "31ce37fe365b3dc204300a3e4c396ad333ed0556 ; touch #{touchdir}/uh_oh", "31ce37fe365b3dc204300a3e4c396ad333ed0556", nil)
      assert !File.exists?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'minimum' parameter of find_commit_range is exploitable"
      assert_equal [], a

      # invalid input to minimum
      a = Commit.find_commit_range('active/foo', "$(uname>#{touchdir}/uh_oh)", "31ce37fe365b3dc204300a3e4c396ad333ed0556", nil)
      assert !File.exists?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'minimum' parameter of find_commit_range is exploitable"
      assert_equal [], a

      # invalid input to 'excludes'
      # complains "fatal: bad object 077ba2ad3ea24a929091a9e6ce545c93199b8e57"
      a = Commit.find_commit_range('active/foo', "31ce37fe365b3dc204300a3e4c396ad333ed0556", "077ba2ad3ea24a929091a9e6ce545c93199b8e57", ["4fe459abe02d9b365932b8f5dc419439ab4e2577 ; touch #{touchdir}/uh_oh"])
      assert !File.exists?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'excludes' parameter of find_commit_range is exploitable"
      assert_equal [], a

      # invalid input to 'excludes'
      # complains "fatal: bad object 077ba2ad3ea24a929091a9e6ce545c93199b8e57"
      a = Commit.find_commit_range('active/foo', "31ce37fe365b3dc204300a3e4c396ad333ed0556", "077ba2ad3ea24a929091a9e6ce545c93199b8e57", ["$(uname>#{touchdir}/uh_oh)"])
      assert !File.exists?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'excludes' parameter of find_commit_range is exploitable"
      assert_equal [], a
    end
  end
end
