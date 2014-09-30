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
$wrong_way_merge_master = /Merge( remote-tracking)? branch '([^\/]+\/)?master' into/
$merge_master = /Merge branch '[^']+'((?! into)| into master)/
$pull_merge = /Merge branch 'master' of /
$refs_or_closes_or_no_issue = /(refs #|closes #|fixes #|no issue #)/i

# enforced custom commit message format
def check_message_format
  all_revs    = `git rev-list --first-parent #{$oldrev}..#{$newrev}`.split("\n")
  merge_revs  = `git rev-list --first-parent --min-parents=2 #{$oldrev}..#{$newrev}`.split("\n")
  # single_revs = `git rev-list --first-parent --max-parents=1 #{$oldrev}..#{$newrev}`.split("\n")
  broken = false
  no_ff = false

  merge_revs.each do |rev|
    message = `git cat-file commit #{rev} | sed '1,/^$/d'`
    if $wrong_way_merge_master.match(message)
      puts "\n[POLICY] Only non-fast-forward merges into master are allowed. Please"
      puts "reset your master branch:"
      puts "  git reset --hard origin/master"
      puts "and then merge your branch with the --no-ff option:"
      puts "  git merge your-branch --no-ff\n"
      puts "Remember to add a reference to an issue number in the merge commit!\n"
      puts "\n******************************************************************\n"
      puts "\nOffending commit: #{rev}\n"
      puts "\nOffending commit message:\n"
      puts message
      puts "\n******************************************************************\n"
      puts "\n\n"
      broken = true
      no_ff = true
    elsif $pull_merge.match(message)
      puts "\n[POLICY] This appears to be a git pull merge of remote master into local"
      puts "master.  In order to maintain a linear first-parent history of master,"
      puts "please reset your branch and remerge or rebase using the latest master.\n"
      puts "\n******************************************************************\n"
      puts "\nOffending commit: #{rev}\n"
      puts "\nOffending commit message:\n\n"
      puts message
      puts "\n******************************************************************\n"
      puts "\n\n"
      broken = true
    elsif not $merge_master.match(message) and not
      puts "\n[POLICY] This does not appear to be a merge of a feature"
      puts "branch into master.  Merges must follow the format"
      puts "\"Merge branch 'feature-branch'\".\n"
      puts "\n******************************************************************\n"
      puts "\nOffending commit: #{rev}\n"
      puts "\nOffending commit message:\n\n"
      puts message
      puts "\n******************************************************************\n"
      puts "\n\n"
      broken = true
    end
  end

  all_revs.each do |rev|
    message = `git cat-file commit #{rev} | sed '1,/^$/d'`
    if $broken_commit_message.match(message)
      puts "\n[POLICY] Rejected broken commit message for including boilerplate"
      puts "instruction text.\n"
      puts "\n******************************************************************\n"
      puts "\nOffending commit: #{rev}\n"
      puts "\nOffending commit message:\n\n"
      puts message
      puts "\n******************************************************************\n"
      puts "\n\n"
      broken = true
    end

    # Do not test when the commit is a no_ff merge (which will be rejected), because
    # this test will complain about *every* commit in the merge otherwise, obscuring
    # the real reason for the rejection (the no_ff merge)
    if not no_ff and not $refs_or_closes_or_no_issue.match(message)
      puts "\n[POLICY] All commits to master must include an issue using \"refs #\" or"
      puts "\"closes #\", or specify \"no issue #\"\n"
      puts "\n******************************************************************\n"
      puts "\nOffending commit: #{rev}\n"
      puts "\nOffending commit message:\n\n"
      puts message
      puts "\n******************************************************************\n"
      puts "\n\n"
      broken = true
    end
  end

  if broken
    exit 1
  end
end

check_message_format
