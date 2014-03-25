class Commit < ActiveRecord::Base
  require 'shellwords'

  def self.find_commit_range(current_user, repository, minimum, maximum, exclude)
    # disallow starting with '-' so verision strings can't be interpreted as command line options
    valid_pattern = /[A-Za-z0-9_][A-Za-z0-9_-]/
    if (minimum and !minimum.match valid_pattern) or !maximum.match valid_pattern
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

        # If not found or string is invalid, nothing else to do
        next if !max_hash or !max_hash.match valid_pattern

        resolved_exclude = nil
        if exclude
          resolved_exclude = []
          exclude.each do |e|
            if e.match valid_pattern
              IO.foreach("|git rev-list --max-count=1 #{e}") do |line|
                resolved_exclude.push(line.strip)
              end
            end
          end
        end

        if minimum
          # Get the commit hash for the lower bound
          min_hash = nil
          IO.foreach("|git rev-list --max-count=1 #{minimum}") do |line|
            min_hash = line.strip
          end

          # If not found or string is invalid, nothing else to do
          next if !min_hash or !min_hash.match valid_pattern

          # Now find all commits between them
          #puts "git rev-list #{min_hash}..#{max_hash}"
          IO.foreach("|git rev-list #{min_hash}..#{max_hash}") do |line|
            hash = line.strip
            commits.push(hash) if !resolved_exclude or !resolved_exclude.include? hash
          end

          commits.push(min_hash) if !resolved_exclude or !resolved_exclude.include? min_hash
        else
          commits.push(max_hash) if !resolved_exclude or !resolved_exclude.include? max_hash
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
