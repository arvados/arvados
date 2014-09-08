if not File.exists?('/usr/bin/git') then
  STDERR.puts "\nGit binary not found, aborting. Please install git and run gem build from a checked out copy of the git repository.\n\n"
  exit
end

Gem::Specification.new do |s|
  s.name        = 'arvados'
  s.version     = '0.1.' + `/usr/bin/git log --pretty=format:'%ci' -n 1`[0..18].gsub(/[ \-:]/,'')
  s.date        = `/usr/bin/git log --pretty=format:'%ci' -n 1`[0..9]
  s.summary     = "Arvados SDK Ruby"
  s.description = "This is the Arvados SDK gem, git revision " + `/usr/bin/git log --pretty=format:'%H' -n 1`
  s.authors     = ["Arvados Authors"]
  s.email       = 'gem-dev@curoverse.com'
  s.licenses    = ['Apache License, Version 2.0']
  s.files       = ["lib/arvados.rb", "lib/arvados/keep.rb"]
  s.required_ruby_version = '>= 2.1.0'
  s.add_dependency('google-api-client', '~> 0.6.3')
  s.add_dependency('activesupport', '>= 3.2.13')
  s.add_dependency('json', '>= 1.7.7')
  s.add_dependency('andand')
  s.add_runtime_dependency('jwt', '>= 0.1.5', '< 1.0.0')
  s.homepage    =
    'http://arvados.org'
end
