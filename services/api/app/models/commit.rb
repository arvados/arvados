class Commit < ActiveRecord::Base
  require 'shellwords'

  # Make sure the specified commit really exists, and return the full
  # sha1 commit hash.
  #
  # Accepts anything "git rev-list" accepts, optionally (and
  # preferably) preceded by "repo_name:".
  #
  # Examples: "1234567", "master", "apps:1234567", "apps:master",
  # "apps:HEAD"

  # def self.find_by_commit_ish(commit_ish)
  #   if only_valid_chars.match(commit_ish)       
  #     logger.warn "find_by_commit_ish called with string containing invalid characters: '#{commit_ish}'"
  #     return nil
  #   end

  #   want_repo = nil
  #   if commit_ish.index(':')
  #     want_repo, commit_ish = commit_ish.split(':',2)
  #   end
  #   repositories.each do |repo_name, repo|
  #     next if want_repo and want_repo != repo_name
  #     ENV['GIT_DIR'] = repo[:git_dir]
  #     # we're passing user input to a command line, this is a potential a security hole but I am reasonably confident that shellescape sanitizes the input adequately
  #     IO.foreach("|git rev-list --max-count=1 --format=oneline 'origin/'#{commit_ish.shellescape} 2>/dev/null || git rev-list --max-count=1 --format=oneline ''#{commit_ish.shellescape}") do |line|
  #       sha1, message = line.strip.split " ", 2
  #       next if sha1.length != 40
  #       begin
  #         Commit.find_or_create_by_repository_name_and_sha1_and_message(repo_name, sha1, message[0..254])
  #       rescue
  #         logger.warn "find_or_create failed: repo_name #{repo_name} sha1 #{sha1} message #{message[0..254]}"
  #         # Ignore cache failure. Commit is real. We should proceed.
  #       end
  #       return sha1
  #     end
  #   end
  #   nil
  # end

  def self.find_commit_range(current_user, repository, minimum, maximum)
    only_valid_chars = /[^A-Za-z0-9_-]/
    if only_valid_chars.match(minimum) || only_valid_chars.match(maximum) 
      logger.warn "find_commit_range called with string containing invalid characters: '#{minimum}', '#{maximum}'"
      return nil
    end

    if minimum and minimum.empty?
        minimum = nil
    end
    
    if !maximum
      maximum = "HEAD"
    end

    # Get list of actual repository directories under management
    on_disk_repos = repositories

    # Get list of repository objects readable by user
    readable = Repository.readable_by(current_user)

    # filter repository objects on requested repository name
    if repository
      readable = readable.where(name: repository)
    end

    #puts "min #{minimum}"
    #puts "max #{maximum}"
    #puts "rep #{repository}"

    commits = []
    readable.each do |r|
      if on_disk_repos[r.name]
        ENV['GIT_DIR'] = on_disk_repos[r.name][:git_dir]

        #puts "dir #{on_disk_repos[r.name][:git_dir]}"

        # We've filtered for invalid characters, so we can pass the contents of
        # minimum and maximum safely on the command line

        #puts "git rev-list --max-count=1 #{maximum}"

        # Get the commit hash for the upper bound
        max_hash = nil
        IO.foreach("|git rev-list --max-count=1 #{maximum}") do |line|
          max_hash = line.strip
        end

        # If not found, nothing else to do
        next if !max_hash

        if minimum          
          # Get the commit hash for the lower bound
          min_hash = nil
          IO.foreach("|git rev-list --max-count=1 #{minimum}") do |line|
            min_hash = line.strip
          end

          # If not found, nothing else to do
          next if !min_hash
          
          # Now find all commits between them
          #puts "git rev-list #{min_hash}..#{max_hash}"
          IO.foreach("|git rev-list #{min_hash}..#{max_hash}") do |line|
            commits.push(line.strip)
          end

          commits.push(min_hash)
        else
          commits.push(max_hash)
        end
      end
    end

    if !commits or commits.empty?
      nil
    else
      commits
    end
  end

  # Import all commits from configured git directory into the commits
  # database.

  def self.import_all
    repositories.each do |repo_name, repo|
      stat = { true => 0, false => 0 }
      ENV['GIT_DIR'] = repo[:git_dir]
      IO.foreach("|git rev-list --format=oneline --all") do |line|
        sha1, message = line.strip.split " ", 2
        imported = false
        Commit.find_or_create_by_repository_name_and_sha1_and_message(repo_name, sha1, message[0..254]) do
          imported = true
        end
        stat[!!imported] += 1
        if (stat[true] + stat[false]) % 100 == 0
          if $stdout.tty? or ARGV[0] == '-v'
            puts "#{$0} #{$$}: repo #{repo_name} add #{stat[true]} skip #{stat[false]}"
          end
        end
      end
      if $stdout.tty? or ARGV[0] == '-v'
        puts "#{$0} #{$$}: repo #{repo_name} add #{stat[true]} skip #{stat[false]}"
      end
    end
  end

  def self.refresh_repositories
    @repositories = nil
  end

  protected

  def self.repositories
    return @repositories if @repositories

    @repositories = {}
    @gitdirbase = Rails.configuration.git_repositories_dir
    Dir.foreach @gitdirbase do |repo|
      next if repo.match /^\./
      git_dir = File.join(@gitdirbase,
                          repo.match(/\.git$/) ? repo : File.join(repo, '.git'))
      repo_name = repo.sub(/\.git$/, '')
      @repositories[repo_name] = {git_dir: git_dir}
    end

    @repositories
  end
end
