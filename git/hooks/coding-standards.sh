#!/usr/bin/env ruby

# This script can be installed as a git update hook.

# It can also be installed as a gitolite 'hooklet' in the
# hooks/common/update.secondary.d/ directory.

# NOTE: this script runs under the same assumptions as the 'update' hook, so
# the starting directory must be maintained and arguments must be passed on.

$refname = ARGV[0]
$oldrev  = ARGV[1]
$newrev  = ARGV[2]
$user    = ENV['USER']

# Only enforce policy on the master branch
exit 0 if $refname != 'refs/heads/master'

puts "Enforcing Policies... \n(#{$refname}) (#{$oldrev[0,6]}) (#{$newrev[0,6]})"

$regex = /\[ref: (\d+)\]/

$broken_commit_message = /Please enter a commit message to explain why this merge is necessary/

$merge_commit = /Merge branch/
$merge_master_commit = /Merge branch 'master' (of|into)/
$refs_or_closes_found = /(refs #|closes #)/i

# If the next command has output, this is a non-merge commit.
#`git rev-list --max-parents=1 --first-parent #{$oldrev}..#{$newrev}`

# enforced custom commit message format
def check_message_format
  missed_revs = `git rev-list #{$oldrev}..#{$newrev}`.split("\n")
  missed_revs.each do |rev|
    message = `git cat-file commit #{rev} | sed '1,/^$/d'`
    if $broken_commit_message.match(message)
      puts "\n[POLICY] Please avoid broken commit messages, rejected\n\n"
			puts "\n******************************************************************\n"
			puts "\nOffending commit: #{rev}\n\n"
			puts "\nOffending commit message:\n\n"
			puts message
			puts "\n******************************************************************\n"
			puts "\n\n"
      exit 1
    end
		if $merge_commit.match(message) and 
				not $merge_master_commit.match(message) and
				not $refs_or_closes_found.match(message)
			puts "\n[POLICY] Please make sure to refer to an issue in all branch merge commits, rejected\n\n"
			puts "\n******************************************************************\n"
			puts "\nOffending commit: #{rev}\n\n"
			puts "\nOffending commit message:\n\n"
			puts message
			puts "\n******************************************************************\n"
			puts "\n\n"
      exit 1
		end
  end
end

check_message_format

