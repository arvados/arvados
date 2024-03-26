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
  version = version.sub("~dev", ".dev").sub("~rc", ".rc")
  git_timestamp = Time.at(git_timestamp.to_i).utc
ensure
  ENV["GIT_DIR"] = git_dir
  ENV["GIT_WORK_TREE"] = git_work
end

Gem::Specification.new do |s|
  s.name        = 'arvados'
  s.version     = version
  s.date        = git_timestamp.strftime("%Y-%m-%d")
  s.summary     = "Arvados client library"
  s.description = "Arvados client library, git commit #{git_hash}"
  s.authors     = ["Arvados Authors"]
  s.email       = 'packaging@arvados.org'
  s.licenses    = ['Apache-2.0']
  s.files       = ["lib/arvados.rb", "lib/arvados/google_api_client.rb",
                   "lib/arvados/collection.rb", "lib/arvados/keep.rb",
                   "README", "LICENSE-2.0.txt"]
  s.required_ruby_version = '>= 2.7.0'
  s.add_dependency('activesupport', '>= 3')
  s.add_dependency('andand', '~> 1.3', '>= 1.3.3')
  # arvados fork of google-api-client gem with old API and new
  # compatibility fixes, built from ../ruby-google-api-client/
  s.add_dependency('arvados-google-api-client', '>= 0.8.7.5', '< 0.8.8')
  # work around undeclared dependency on i18n in some activesupport 3.x.x:
  s.add_dependency('i18n')
  s.add_dependency('json', '>= 1.7.7', '<3')
  s.add_runtime_dependency('jwt', '<2', '>= 0.1.5')
  s.homepage    =
    'https://arvados.org'
end
