require 'fileutils'
require 'tmpdir'

module GitSetup
  def setup
    @tmpdir = Dir.mktmpdir()
    #puts "setup #{@tmpdir}"
    `cp test/test.git.tar #{@tmpdir} && cd #{@tmpdir} && tar xf test.git.tar`
    Rails.configuration.git_repositories_dir = "#{@tmpdir}/test"
    Commit.refresh_repositories
  end

  def teardown
    #puts "teardown #{@tmpdir}"
    FileUtils.remove_entry @tmpdir, true
  end
end
