# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

if not File.exist?('/usr/bin/git') then
  STDERR.puts "\nGit binary not found, aborting. Please install git and run gem build from a checked out copy of the git repository.\n\n"
  exit
end

git_dir = ENV["GIT_DIR"]
git_work = ENV["GIT_WORK_TREE"]
begin
  ENV["GIT_DIR"] = File.expand_path "#{__dir__}/../../.git"
  ENV["GIT_WORK_TREE"] = File.expand_path "#{__dir__}/../.."
  git_timestamp, git_hash = `git log -n1 --first-parent --format=%ct:%H #{__dir__}`.chomp.split(":")
  if ENV["ARVADOS_BUILDING_VERSION"]
    version = ENV["ARVADOS_BUILDING_VERSION"]
  else
    version = `#{__dir__}/../../build/version-at-commit.sh #{git_hash}`.encode('utf-8').strip
  end
  version = version.sub("~dev", ".dev").sub("~rc", ".rc")
  git_timestamp = Time.at(git_timestamp.to_i).utc
ensure
  ENV["GIT_DIR"] = git_dir
  ENV["GIT_WORK_TREE"] = git_work
end

Gem::Specification.new do |s|
  s.name        = 'arvados-login-sync'
  s.version     = version
  s.date        = git_timestamp.strftime("%Y-%m-%d")
  s.summary     = "Set up local login accounts for Arvados users"
  s.description = "Creates and updates local login accounts for Arvados users. Built from git commit #{git_hash}"
  s.authors     = ["Arvados Authors"]
  s.email       = 'packaging@arvados.org'
  s.licenses    = ['AGPL-3.0']
  s.files       = ["bin/arvados-login-sync", "agpl-3.0.txt"]
  s.executables << "arvados-login-sync"
  s.required_ruby_version = '>= 2.5.0'
  # The minimum version's 'a' suffix is necessary to enable bundler
  # to consider 'pre-release' versions.  See:
  # https://github.com/rubygems/bundler/issues/4340
  s.add_runtime_dependency 'arvados', '~> 2.8.a'
  s.add_runtime_dependency 'launchy', '< 2.5'
  # arvados fork of google-api-client gem with old API and new
  # compatibility fixes, built from ../../sdk/ruby-google-api-client/
  s.add_runtime_dependency('arvados-google-api-client', '>= 0.8.7.5', '< 0.8.9')
  s.homepage    =
    'https://arvados.org'
end
