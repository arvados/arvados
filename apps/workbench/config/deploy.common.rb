before 'deploy:update_code' do
  local_branch = `git branch | egrep '^\\*' | cut -d' ' -f2`.strip
  remote_commit = `git ls-remote '#{fetch(:repository)}' '#{local_branch}'`.strip.split.first
  local_commit = `git show -s --format=format:%H`.strip
  if !local_branch.match(/^(master$|production)/)
    abort 'You cannot deploy unless your branch is called "master" or "production*"'
  end
  if local_commit != remote_commit
    puts "Current branch is #{local_branch}"
    puts "Last commit is #{local_commit} here"
    puts "Last commit is #{remote_commit} on #{local_branch} at #{fetch(:repository)}"
    abort "You cannot deploy unless HEAD = a branch = head of remote branch with same name."
  end
  puts "Setting deploy branch to #{local_branch}"
  set :branch, local_branch
end
