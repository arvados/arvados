Gem::Specification.new do |s|
  s.name        = 'arvados'
  s.version     = '0.1.0'
  s.date        = `/usr/bin/git log --pretty=format:'%ci' -n 1`[0..9]
  s.summary     = "Arvados SDK"
  s.description = "This is the Arvados SDK gem, git revision " + `/usr/bin/git log --pretty=format:'%H' -n 1`
  s.authors     = ["Arvados Authors"]
  s.email       = 'gem-dev@clinicalfuture.com'
  s.licenses    = ['Apache License, Version 2.0']
  s.files       = ["lib/arvados.rb"]
  s.homepage    =
    'http://arvados.org'
end
