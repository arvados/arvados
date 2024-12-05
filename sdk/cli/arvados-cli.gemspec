# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

begin
  git_root = "#{__dir__}/../.."
  git_timestamp, git_hash = IO.popen(
    ["git", "-C", git_root,
     "log", "-n1", "--first-parent", "--format=%ct:%H",
     "--", "build/version-at-commit.sh", "sdk/ruby", "sdk/cli"],
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
  s.required_ruby_version = '>= 2.7.0'
  s.add_runtime_dependency 'arvados', arv_dep_version
  # arvados fork of google-api-client gem with old API and new
  # compatibility fixes, built from ../ruby-google-api-client/
  s.add_runtime_dependency('arvados-google-api-client', '>= 0.8.7.5', '< 0.8.9')
  s.add_runtime_dependency 'activesupport', '>= 3.2.13', '< 8.0'
  s.add_runtime_dependency 'json', '>= 1.7.7', '<3'
  s.add_runtime_dependency 'optimist', '~> 3.0'
  s.add_runtime_dependency 'andand', '~> 1.3', '>= 1.3.3'
  # oj 3.10.9 requires ruby >= 2.4 and arvbox doesn't currently have it because of SSO
  s.add_runtime_dependency 'oj', '< 3.10.9'
  s.add_runtime_dependency 'curb', '~> 0.8'
  s.add_runtime_dependency 'launchy', '< 2.5'
  s.homepage    =
    'https://arvados.org'
end
