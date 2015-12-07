if not File.exists?('/usr/bin/git') then
  STDERR.puts "\nGit binary not found, aborting. Please install git and run gem build from a checked out copy of the git repository.\n\n"
  exit
end

git_timestamp, git_hash = `git log -n1 --first-parent --format=%ct:%H .`.chomp.split(":")
git_timestamp = Time.at(git_timestamp.to_i).utc

Gem::Specification.new do |s|
  s.name        = 'arvados-login-sync'
  s.version     = "0.1.#{git_timestamp.strftime('%Y%m%d%H%M%S')}"
  s.date        = git_timestamp.strftime("%Y-%m-%d")
  s.summary     = "Set up local login accounts for Arvados users"
  s.description = "Creates and updates local login accounts for Arvados users. Built from git commit #{git_hash}"
  s.authors     = ["Arvados Authors"]
  s.email       = 'gem-dev@curoverse.com'
  s.licenses    = ['GNU Affero General Public License, version 3.0']
  s.files       = ["bin/arvados-login-sync", "agpl-3.0.txt"]
  s.executables << "arvados-login-sync"
  s.required_ruby_version = '>= 2.1.0'
  s.add_runtime_dependency 'arvados', '~> 0.1', '>= 0.1.20150615153458'
  s.homepage    =
    'https://arvados.org'
end
