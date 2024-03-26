# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AppVersion
  def self.git(*args, &block)
    IO.popen(["git", "--git-dir", ".git"] + args, "r",
             chdir: Rails.root.join('../..'),
             err: "/dev/null",
             &block)
  end

  def self.forget
    @hash = nil
    @package_version = nil
  end

  # Return abbrev commit hash for current code version: "abc1234", or
  # "abc1234-modified" if there are uncommitted changes. If present,
  # return contents of {root}/git-commit.version instead.
  def self.hash
    if (cached = Rails.configuration.source_version || @hash)
      return cached
    end

    # Read the version from our package's git-commit.version file, if available.
    begin
      @hash = IO.read(Rails.root.join("git-commit.version")).strip
    rescue Errno::ENOENT
    end

    if @hash.nil? or @hash.empty?
      begin
        local_modified = false
        git("status", "--porcelain") do |git_pipe|
          git_pipe.each_line do |_|
            local_modified = true
            # Continue reading the pipe so git doesn't get SIGPIPE.
          end
        end
        if $?.success?
          git("log", "-n1", "--format=%H") do |git_pipe|
            git_pipe.each_line do |line|
              @hash = line.chomp[0...8] + (local_modified ? '-modified' : '')
            end
          end
        end
      rescue SystemCallError
      end
    end

    @hash || "unknown"
  end

  def self.package_version
    if (cached = Rails.configuration.package_version || @package_version)
      return cached
    end

    begin
      @package_version = IO.read(Rails.root.join("package-build.version")).strip
    rescue Errno::ENOENT
      @package_version = "unknown"
    end

    @package_version
  end
end
