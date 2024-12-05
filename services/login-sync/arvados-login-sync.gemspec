# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

begin
  git_root = "#{__dir__}/../.."
  git_timestamp, git_hash = IO.popen(
    ["git", "-C", git_root,
     "log", "-n1", "--first-parent", "--format=%ct:%H",
     "--", "build/version-at-commit.sh", "sdk/ruby", "services/login-sync"],
  ) do |git_log|
    git_log.readline.chomp.split(":")
  end
rescue Errno::ENOENT
  $stderr.puts("failed to get version information: 'git' not found")
  exit 69  # EX_UNAVAILABLE
end

if $? != 0
  $stderr.puts("failed to get version information: 'git log' exited #{$?}")
  exit 65  # EX_DATAERR
end
git_timestamp = Time.at(git_timestamp.to_i).utc
version = ENV["ARVADOS_BUILDING_VERSION"] || IO.popen(
            ["#{git_root}/build/version-at-commit.sh", git_hash],
          ) do |ver_out|
  ver_out.readline.chomp.encode("utf-8")
end
version = version.sub("~dev", ".dev").sub("~rc", ".rc")
arv_dep_version = if dev_index = (version =~ /\.dev/)
                    "~> #{version[...dev_index]}.a"
                  else
                    "= #{version}"
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
  s.add_runtime_dependency 'arvados', arv_dep_version
  s.add_runtime_dependency 'launchy', '< 2.5'
  # arvados fork of google-api-client gem with old API and new
  # compatibility fixes, built from ../../sdk/ruby-google-api-client/
  s.add_runtime_dependency('arvados-google-api-client', '>= 0.8.7.5', '< 0.8.9')
  s.homepage    =
    'https://arvados.org'
end
