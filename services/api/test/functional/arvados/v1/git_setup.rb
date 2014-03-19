require 'fileutils'
require 'tmpdir'

# Commit log for test.git.tar
# master is the main branch
# b1 is a branch off of master
# tag1 is a tag
#
# 1de84a8 * b1
# 077ba2a * master
# 4fe459a * tag1
# 31ce37f * foo

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
