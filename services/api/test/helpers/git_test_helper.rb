require 'fileutils'
require 'tmpdir'

# Commit log for "foo" repository in test.git.tar
# master is the main branch
# b1 is a branch off of master
# tag1 is a tag
#
# 1de84a8 * b1
# 077ba2a * master
# 4fe459a * tag1
# 31ce37f * foo

module GitTestHelper
  def self.included base
    base.setup do
      @tmpdir = Dir.mktmpdir()
      system("tar", "-xC", @tmpdir, "-f", "test/test.git.tar")
      Rails.configuration.git_repositories_dir = "#{@tmpdir}/test"
      Commit.refresh_repositories
    end

    base.teardown do
      FileUtils.remove_entry @tmpdir, true
      Commit.refresh_repositories
    end
  end
end
