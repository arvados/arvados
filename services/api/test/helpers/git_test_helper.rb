# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'fileutils'
require 'tmpdir'

# Commit log for "foo" repository in test.git.tar
# main is the main branch
# b1 is a branch off of main 
# tag1 is a tag
#
# 1de84a8 * b1
# 077ba2a * main
# 4fe459a * tag1
# 31ce37f * foo

module GitTestHelper
  def self.included base
    base.setup do
      # Extract the test repository data into the default test
      # environment's Rails.configuration.Git.Repositories. (We
      # don't use that config setting here, though: it doesn't seem
      # worth the risk of stepping on a real git repo root.)
      @tmpdir = Rails.root.join 'tmp', 'git'
      FileUtils.mkdir_p @tmpdir
      system("tar", "-xC", @tmpdir.to_s, "-f", "test/test.git.tar")
      Rails.configuration.Git.Repositories = "#{@tmpdir}/test"
      Rails.configuration.Containers.JobsAPI.GitInternalDir = "#{@tmpdir}/internal.git"
    end

    base.teardown do
      FileUtils.remove_entry CommitsHelper.cache_dir_base, true
      FileUtils.mkdir_p @tmpdir
      system("tar", "-xC", @tmpdir.to_s, "-f", "test/test.git.tar")
    end
  end

  def internal_tag tag
    IO.read "|git --git-dir #{Rails.configuration.Containers.JobsAPI.GitInternalDir.shellescape} log --format=format:%H -n1 #{tag.shellescape}"
  end

  # Intercept fetch_remote_repository and fetch from a specified url
  # or local fixture instead of the remote url requested. fakeurl can
  # be a url (probably starting with file:///) or the name of a
  # fixture (as a symbol)
  def fetch_remote_from_local_repo url, fakeurl
    if fakeurl.is_a? Symbol
      fakeurl = 'file://' + repositories(fakeurl).server_path
    end
    CommitsHelper.expects(:fetch_remote_repository).once.with do |gitdir, giturl|
      if giturl == url
        CommitsHelper.unstub(:fetch_remote_repository)
        CommitsHelper.fetch_remote_repository gitdir, fakeurl
        true
      end
    end
  end
end
