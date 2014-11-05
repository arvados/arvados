if not File.exists?('/usr/bin/git') then
  STDERR.puts "\nGit binary not found, aborting. Please install git and run gem build from a checked out copy of the git repository.\n\n"
  exit
end

git_timestamp, git_hash = `git log -n1 --first-parent --format=%ct:%H .`.chomp.split(":")
git_timestamp = Time.at(git_timestamp.to_i).utc

Gem::Specification.new do |s|
  s.name        = 'arvados-cli'
  s.version     = "0.1.#{git_timestamp.strftime('%Y%m%d%H%M%S')}"
  s.date        = git_timestamp.strftime("%Y-%m-%d")
  s.summary     = "Arvados CLI tools"
  s.description = "Arvados command line tools, git commit #{git_hash}"
  s.authors     = ["Arvados Authors"]
  s.email       = 'gem-dev@curoverse.com'
  #s.bindir      = '.'
  s.licenses    = ['Apache License, Version 2.0']
  s.files       = ["bin/arv","bin/arv-run-pipeline-instance","bin/arv-crunch-job","bin/arv-tag","bin/crunch-job"]
  s.executables << "arv"
  s.executables << "arv-run-pipeline-instance"
  s.executables << "arv-crunch-job"
  s.executables << "arv-tag"
  s.required_ruby_version = '>= 2.1.0'
  s.add_runtime_dependency 'arvados', '~> 0.1', '>= 0.1.0'
  s.add_runtime_dependency 'google-api-client', '~> 0.6.3', '>= 0.6.3'
  s.add_runtime_dependency 'activesupport', '~> 3.2', '>= 3.2.13'
  s.add_runtime_dependency 'json', '~> 1.7', '>= 1.7.7'
  s.add_runtime_dependency 'trollop', '~> 2.0'
  s.add_runtime_dependency 'andand', '~> 1.3', '>= 1.3.3'
  s.add_runtime_dependency 'oj', '~> 2.0', '>= 2.0.3'
  s.add_runtime_dependency 'curb', '~> 0.8'
  s.add_runtime_dependency('jwt', '>= 0.1.5', '< 1.0.0')
  s.homepage    =
    'https://arvados.org'
end
