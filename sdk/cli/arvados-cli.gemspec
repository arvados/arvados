# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

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
  git_timestamp = Time.at(git_timestamp.to_i).utc
ensure
  ENV["GIT_DIR"] = git_dir
  ENV["GIT_WORK_TREE"] = git_work
end

Gem::Specification.new do |s|
  s.name        = 'arvados-cli'
  s.version     = version
  s.date        = git_timestamp.strftime("%Y-%m-%d")
  s.summary     = "Arvados CLI tools"
  s.description = "Arvados command line tools, git commit #{git_hash}"
  s.authors     = ["Arvados Authors"]
  s.email       = 'packaging@arvados.org'
  #s.bindir      = '.'
  s.licenses    = ['Apache-2.0']
  s.files       = ["bin/arv", "bin/arv-tag", "LICENSE-2.0.txt"]
  s.executables << "arv"
  s.executables << "arv-tag"
  s.required_ruby_version = '>= 2.1.0'
  s.add_runtime_dependency 'arvados', '>= 1.4.1.20190320201707'
  # Our google-api-client dependency used to be < 0.9, but that could be
  # satisfied by the buggy 0.9.pre*.  https://dev.arvados.org/issues/9213
  s.add_runtime_dependency 'arvados-google-api-client', '~> 0.6', '>= 0.6.3', '<0.8.9'
  s.add_runtime_dependency 'activesupport', '>= 3.2.13', '< 5.3'
  s.add_runtime_dependency 'json', '>= 1.7.7', '<3'
  s.add_runtime_dependency 'optimist', '~> 3.0'
  s.add_runtime_dependency 'andand', '~> 1.3', '>= 1.3.3'
  # oj 3.10.9 requires ruby >= 2.4 and arvbox doesn't currently have it because of SSO
  s.add_runtime_dependency 'oj', '< 3.10.9'
  s.add_runtime_dependency 'curb', '~> 0.8'
  s.add_runtime_dependency 'launchy', '< 2.5'
  # arvados-google-api-client 0.8.7.2 is incompatible with faraday 0.16.2
  s.add_dependency('faraday', '< 0.16')
  s.homepage    =
    'https://arvados.org'
end
