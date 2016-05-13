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
  s.files       = ["lib/arvados.rb", "lib/arvados/google_api_client.rb",
                   "lib/arvados/collection.rb", "lib/arvados/keep.rb",
                   "README", "LICENSE-2.0.txt"]
  s.required_ruby_version = '>= 1.8.7'
  # activesupport <4.2.6 only because https://dev.arvados.org/issues/8222
  s.add_dependency('activesupport', '>= 3', '< 4.2.6')
  s.add_dependency('andand', '~> 1.3', '>= 1.3.3')
  # Our google-api-client dependency used to be < 0.9, but that could be
  # satisfied by the buggy 0.9.pre*.  https://dev.arvados.org/issues/9213
  s.add_dependency('google-api-client', '>= 0.7', '< 0.8.9')
  # work around undeclared dependency on i18n in some activesupport 3.x.x:
  s.add_dependency('i18n', '~> 0')
  s.add_dependency('json', '~> 1.7', '>= 1.7.7')
  s.add_runtime_dependency('jwt', '<2', '>= 0.1.5')
  s.homepage    =
    'https://arvados.org'
end
