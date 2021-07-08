# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

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
      CommitsHelper::find_commit_range 'foo', nil, 'main', []
    end
  end

  def must_pipe(cmd)
    begin
      return IO.read("|#{cmd}")
    ensure
      assert $?.success?
    end
  end

  [
   'https://github.com/arvados/arvados.git',
   'http://github.com/arvados/arvados.git',
   'git://github.com/arvados/arvados.git',
  ].each do |url|
    test "find_commit_range uses fetch_remote_repository to get #{url}" do
      fake_gitdir = repositories(:foo).server_path
      CommitsHelper::expects(:cache_dir_for).once.with(url).returns fake_gitdir
      CommitsHelper::expects(:fetch_remote_repository).once.with(fake_gitdir, url).returns true
      c = CommitsHelper::find_commit_range url, nil, 'main', []
      refute_empty c
    end
  end

  [
   'bogus/repo',
   '/bogus/repo',
   '/not/allowed/.git',
   'file:///not/allowed.git',
   'git.arvados.org/arvados.git',
   'github.com/arvados/arvados.git',
  ].each do |url|
    test "find_commit_range skips fetch_remote_repository for #{url}" do
      CommitsHelper::expects(:fetch_remote_repository).never
      assert_raises ArgumentError do
        CommitsHelper::find_commit_range url, nil, 'main', []
      end
    end
  end

  test 'fetch_remote_repository does not leak commits across repositories' do
    url = "http://localhost:1/fake/fake.git"
    fetch_remote_from_local_repo url, :foo
    c = CommitsHelper::find_commit_range url, nil, 'main', []
    assert_equal ['077ba2ad3ea24a929091a9e6ce545c93199b8e57'], c

    url = "http://localhost:2/fake/fake.git"
    fetch_remote_from_local_repo url, 'file://' + File.expand_path('../../.git', Rails.root)
    c = CommitsHelper::find_commit_range url, nil, '077ba2ad3ea24a929091a9e6ce545c93199b8e57', []
    assert_equal [], c
  end

  test 'tag_in_internal_repository creates and updates tags in internal.git' do
    authorize_with :active
    gitint = "git --git-dir #{Rails.configuration.Containers.JobsAPI.GitInternalDir}"
    IO.read("|#{gitint} tag -d testtag 2>/dev/null") # "no such tag", fine
    assert_match(/^fatal: /, IO.read("|#{gitint} show testtag 2>&1"))
    refute $?.success?
    CommitsHelper::tag_in_internal_repository 'active/foo', '31ce37fe365b3dc204300a3e4c396ad333ed0556', 'testtag'
    assert_match(/^commit 31ce37f/, IO.read("|#{gitint} show testtag"))
    assert $?.success?
  end

  def with_foo_repository
    Dir.chdir("#{Rails.configuration.Git.Repositories}/#{repositories(:foo).uuid}") do
      must_pipe("git checkout main 2>&1")
      yield
    end
  end

  test 'tag_in_internal_repository, new non-tip sha1 in local repo' do
    tag = "tag#{rand(10**10)}"
    sha1 = nil
    with_foo_repository do
      must_pipe("git checkout -b branch-#{rand(10**10)} 2>&1")
      must_pipe("echo -n #{tag.shellescape} >bar")
      must_pipe("git add bar")
      must_pipe("git -c user.email=x@x -c user.name=X commit -m -")
      sha1 = must_pipe("git log -n1 --format=%H").strip
      must_pipe("git rm bar")
      must_pipe("git -c user.email=x@x -c user.name=X commit -m -")
    end
    CommitsHelper::tag_in_internal_repository 'active/foo', sha1, tag
    gitint = "git --git-dir #{Rails.configuration.Containers.JobsAPI.GitInternalDir.shellescape}"
    assert_match(/^commit /, IO.read("|#{gitint} show #{tag.shellescape}"))
    assert $?.success?
  end

  test 'tag_in_internal_repository, new unreferenced sha1 in local repo' do
    tag = "tag#{rand(10**10)}"
    sha1 = nil
    with_foo_repository do
      must_pipe("echo -n #{tag.shellescape} >bar")
      must_pipe("git add bar")
      must_pipe("git -c user.email=x@x -c user.name=X commit -m -")
      sha1 = must_pipe("git log -n1 --format=%H").strip
      must_pipe("git reset --hard HEAD^")
    end
    CommitsHelper::tag_in_internal_repository 'active/foo', sha1, tag
    gitint = "git --git-dir #{Rails.configuration.Containers.JobsAPI.GitInternalDir.shellescape}"
    assert_match(/^commit /, IO.read("|#{gitint} show #{tag.shellescape}"))
    assert $?.success?
  end

  # In active/shabranchnames, "7387838c69a21827834586cc42b467ff6c63293b" is
  # both a commit hash, and the name of a branch that begins from that same
  # commit.
  COMMIT_BRANCH_NAME = "7387838c69a21827834586cc42b467ff6c63293b"
  # A commit that appears in the branch after 7387838c.
  COMMIT_BRANCH_COMMIT_2 = "abec49829bf1758413509b7ffcab32a771b71e81"
  # "738783" is another branch that starts from the above commit.
  SHORT_COMMIT_BRANCH_NAME = COMMIT_BRANCH_NAME[0, 6]
  # A commit that appears in branch 738783 after 7387838c.
  SHORT_BRANCH_COMMIT_2 = "77e1a93093663705a63bb4d505698047e109dedd"

  test "find_commit_range min_version prefers commits over branch names" do
    assert_equal([COMMIT_BRANCH_NAME],
                 CommitsHelper::find_commit_range("active/shabranchnames",
                                          COMMIT_BRANCH_NAME, nil, nil))
  end

  test "find_commit_range max_version prefers commits over branch names" do
    assert_equal([COMMIT_BRANCH_NAME],
                 CommitsHelper::find_commit_range("active/shabranchnames",
                                          nil, COMMIT_BRANCH_NAME, nil))
  end

  test "find_commit_range min_version with short branch name" do
    assert_equal([SHORT_BRANCH_COMMIT_2],
                 CommitsHelper::find_commit_range("active/shabranchnames",
                                          SHORT_COMMIT_BRANCH_NAME, nil, nil))
  end

  test "find_commit_range max_version with short branch name" do
    assert_equal([SHORT_BRANCH_COMMIT_2],
                 CommitsHelper::find_commit_range("active/shabranchnames",
                                          nil, SHORT_COMMIT_BRANCH_NAME, nil))
  end

  test "find_commit_range min_version with disambiguated branch name" do
    assert_equal([COMMIT_BRANCH_COMMIT_2],
                 CommitsHelper::find_commit_range("active/shabranchnames",
                                          "heads/#{COMMIT_BRANCH_NAME}",
                                          nil, nil))
  end

  test "find_commit_range max_version with disambiguated branch name" do
    assert_equal([COMMIT_BRANCH_COMMIT_2],
                 CommitsHelper::find_commit_range("active/shabranchnames", nil,
                                          "heads/#{COMMIT_BRANCH_NAME}", nil))
  end

  test "find_commit_range min_version with unambiguous short name" do
    assert_equal([COMMIT_BRANCH_NAME],
                 CommitsHelper::find_commit_range("active/shabranchnames",
                                          COMMIT_BRANCH_NAME[0..-2], nil, nil))
  end

  test "find_commit_range max_version with unambiguous short name" do
    assert_equal([COMMIT_BRANCH_NAME],
                 CommitsHelper::find_commit_range("active/shabranchnames", nil,
                                          COMMIT_BRANCH_NAME[0..-2], nil))
  end

  test "find_commit_range laundry list" do
    authorize_with :active

    # single
    a = CommitsHelper::find_commit_range('active/foo', nil, '31ce37fe365b3dc204300a3e4c396ad333ed0556', nil)
    assert_equal ['31ce37fe365b3dc204300a3e4c396ad333ed0556'], a

    #test "test_branch1" do
    a = CommitsHelper::find_commit_range('active/foo', nil, 'main', nil)
    assert_includes(a, '077ba2ad3ea24a929091a9e6ce545c93199b8e57')

    #test "test_branch2" do
    a = CommitsHelper::find_commit_range('active/foo', nil, 'b1', nil)
    assert_equal ['1de84a854e2b440dc53bf42f8548afa4c17da332'], a

    #test "test_branch3" do
    a = CommitsHelper::find_commit_range('active/foo', nil, 'HEAD', nil)
    assert_equal ['1de84a854e2b440dc53bf42f8548afa4c17da332'], a

    #test "test_single_revision_repo" do
    a = CommitsHelper::find_commit_range('active/foo', nil, '31ce37fe365b3dc204300a3e4c396ad333ed0556', nil)
    assert_equal ['31ce37fe365b3dc204300a3e4c396ad333ed0556'], a
    a = CommitsHelper::find_commit_range('arvados', nil, '31ce37fe365b3dc204300a3e4c396ad333ed0556', nil)
    assert_equal [], a

    #test "test_multi_revision" do
    # complains "fatal: bad object 077ba2ad3ea24a929091a9e6ce545c93199b8e57"
    a = CommitsHelper::find_commit_range('active/foo', '31ce37fe365b3dc204300a3e4c396ad333ed0556', '077ba2ad3ea24a929091a9e6ce545c93199b8e57', nil)
    assert_equal ['077ba2ad3ea24a929091a9e6ce545c93199b8e57', '4fe459abe02d9b365932b8f5dc419439ab4e2577', '31ce37fe365b3dc204300a3e4c396ad333ed0556'], a

    #test "test_tag" do
    # complains "fatal: ambiguous argument 'tag1': unknown revision or path
    # not in the working tree."
    a = CommitsHelper::find_commit_range('active/foo', 'tag1', 'main', nil)
    assert_equal ['077ba2ad3ea24a929091a9e6ce545c93199b8e57', '4fe459abe02d9b365932b8f5dc419439ab4e2577'], a

    #test "test_multi_revision_exclude" do
    a = CommitsHelper::find_commit_range('active/foo', '31ce37fe365b3dc204300a3e4c396ad333ed0556', '077ba2ad3ea24a929091a9e6ce545c93199b8e57', ['4fe459abe02d9b365932b8f5dc419439ab4e2577'])
    assert_equal ['077ba2ad3ea24a929091a9e6ce545c93199b8e57', '31ce37fe365b3dc204300a3e4c396ad333ed0556'], a

    #test "test_multi_revision_tagged_exclude" do
    # complains "fatal: bad object 077ba2ad3ea24a929091a9e6ce545c93199b8e57"
    a = CommitsHelper::find_commit_range('active/foo', '31ce37fe365b3dc204300a3e4c396ad333ed0556', '077ba2ad3ea24a929091a9e6ce545c93199b8e57', ['tag1'])
    assert_equal ['077ba2ad3ea24a929091a9e6ce545c93199b8e57', '31ce37fe365b3dc204300a3e4c396ad333ed0556'], a

    Dir.mktmpdir do |touchdir|
      # invalid input to maximum
      a = CommitsHelper::find_commit_range('active/foo', nil, "31ce37fe365b3dc204300a3e4c396ad333ed0556 ; touch #{touchdir}/uh_oh", nil)
      assert !File.exist?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'maximum' parameter of find_commit_range is exploitable"
      assert_equal [], a

      # invalid input to maximum
      a = CommitsHelper::find_commit_range('active/foo', nil, "$(uname>#{touchdir}/uh_oh)", nil)
      assert !File.exist?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'maximum' parameter of find_commit_range is exploitable"
      assert_equal [], a

      # invalid input to minimum
      a = CommitsHelper::find_commit_range('active/foo', "31ce37fe365b3dc204300a3e4c396ad333ed0556 ; touch #{touchdir}/uh_oh", "31ce37fe365b3dc204300a3e4c396ad333ed0556", nil)
      assert !File.exist?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'minimum' parameter of find_commit_range is exploitable"
      assert_equal [], a

      # invalid input to minimum
      a = CommitsHelper::find_commit_range('active/foo', "$(uname>#{touchdir}/uh_oh)", "31ce37fe365b3dc204300a3e4c396ad333ed0556", nil)
      assert !File.exist?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'minimum' parameter of find_commit_range is exploitable"
      assert_equal [], a

      # invalid input to 'excludes'
      # complains "fatal: bad object 077ba2ad3ea24a929091a9e6ce545c93199b8e57"
      a = CommitsHelper::find_commit_range('active/foo', "31ce37fe365b3dc204300a3e4c396ad333ed0556", "077ba2ad3ea24a929091a9e6ce545c93199b8e57", ["4fe459abe02d9b365932b8f5dc419439ab4e2577 ; touch #{touchdir}/uh_oh"])
      assert !File.exist?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'excludes' parameter of find_commit_range is exploitable"
      assert_equal [], a

      # invalid input to 'excludes'
      # complains "fatal: bad object 077ba2ad3ea24a929091a9e6ce545c93199b8e57"
      a = CommitsHelper::find_commit_range('active/foo', "31ce37fe365b3dc204300a3e4c396ad333ed0556", "077ba2ad3ea24a929091a9e6ce545c93199b8e57", ["$(uname>#{touchdir}/uh_oh)"])
      assert !File.exist?("#{touchdir}/uh_oh"), "#{touchdir}/uh_oh should not exist, 'excludes' parameter of find_commit_range is exploitable"
      assert_equal [], a
    end
  end
end
