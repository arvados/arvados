# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Usage:
#
# x = CommitAncestor.find_or_create_by_descendant_and_ancestor(a, b)
# "b is an ancestor of a" if x.is
#

class CommitAncestor < ActiveRecord::Base
  before_create :ask_git_whether_is

  class CommitNotFoundError < ArgumentError
  end

  protected

  def ask_git_whether_is
    @gitdirbase = Rails.configuration.git_repositories_dir
    self.is = nil
    Dir.foreach @gitdirbase do |repo|
      next if repo.match(/^\./)
      git_dir = repo.match(/\.git$/) ? repo : File.join(repo, '.git')
      repo_name = repo.sub(/\.git$/, '')
      ENV['GIT_DIR'] = File.join(@gitdirbase, git_dir)
      IO.foreach("|git rev-list --format=oneline '#{self.descendant.gsub(/[^0-9a-f]/,"")}'") do |line|
        self.is = false
        sha1, _ = line.strip.split(" ", 2)
        if sha1 == self.ancestor
          self.is = true
          break
        end
      end
      if !self.is.nil?
        self.repository_name = repo_name
        break
      end
    end
    if self.is.nil?
      raise CommitNotFoundError.new "Specified commit was not found"
    end
  end
end
