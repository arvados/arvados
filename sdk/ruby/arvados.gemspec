if not File.exists?('/usr/bin/git') then
  STDERR.puts "\nGit binary not found, aborting. Please install git and run gem build from a checked out copy of the git repository.\n\n"
  exit
end

git_timestamp, git_hash = `git log -n1 --first-parent --format=%ct:%H .`.chomp.split(":")
git_timestamp = Time.at(git_timestamp.to_i).utc

Gem::Specification.new do |s|
  s.name        = 'arvados'
  s.version     = "0.1.#{git_timestamp.strftime('%Y%m%d%H%M%S')}"
  s.date        = git_timestamp.strftime("%Y-%m-%d")
  s.summary     = "Arvados client library"
  s.description = "Arvados client library, git commit #{git_hash}"
  s.authors     = ["Arvados Authors"]
  s.email       = 'gem-dev@curoverse.com'
  s.licenses    = ['Apache License, Version 2.0']
  s.files       = ["lib/arvados.rb", "lib/arvados/keep.rb"]
  s.required_ruby_version = '>= 2.1.0'
  s.add_dependency('google-api-client', '~> 0.6.3', '>= 0.6.3')
  s.add_dependency('activesupport', '~> 3.2', '>= 3.2.13')
  s.add_dependency('json', '~> 1.7', '>= 1.7.7')
  s.add_dependency('andand', '~> 1.3', '>= 1.3.3')
  s.add_runtime_dependency('jwt', '>= 0.1.5', '< 1.0.0')
  s.homepage    =
    'https://arvados.org'
end
