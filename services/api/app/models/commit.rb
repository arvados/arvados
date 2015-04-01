class Commit < ActiveRecord::Base
  require 'shellwords'

  def self.git_check_ref_format(e)
    if !e or e.empty? or e[0] == '-' or e[0] == '$'
      # definitely not valid
      false
    else
      `git check-ref-format --allow-onelevel #{e.shellescape}`
      $?.success?
    end
  end

  def self.find_commit_range(current_user, repository, minimum, maximum, exclude)
    if minimum and minimum.empty?
      minimum = nil
    end

    if minimum and !git_check_ref_format(minimum)
      logger.warn "find_commit_range called with invalid minimum revision: '#{minimum}'"
      return nil
    end

    if maximum and !git_check_ref_format(maximum)
      logger.warn "find_commit_range called with invalid maximum revision: '#{maximum}'"
      return nil
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

    commits = []
    readable.each do |r|
      if on_disk_repos[r.name]
        ENV['GIT_DIR'] = on_disk_repos[r.name][:git_dir]

        # We've filtered for invalid characters, so we can pass the contents of
        # minimum and maximum safely on the command line

        # Get the commit hash for the upper bound
        max_hash = nil
        IO.foreach("|git rev-list --max-count=1 #{maximum.shellescape} --") do |line|
          max_hash = line.strip
        end

        # If not found or string is invalid, nothing else to do
        next if !max_hash or !git_check_ref_format(max_hash)

        resolved_exclude = nil
        if exclude
          resolved_exclude = []
          exclude.each do |e|
            if git_check_ref_format(e)
              IO.foreach("|git rev-list --max-count=1 #{e.shellescape} --") do |line|
                resolved_exclude.push(line.strip)
              end
            else
              logger.warn "find_commit_range called with invalid exclude invalid characters: '#{exclude}'"
              return nil
            end
          end
        end

        if minimum
          # Get the commit hash for the lower bound
          min_hash = nil
          IO.foreach("|git rev-list --max-count=1 #{minimum.shellescape} --") do |line|
            min_hash = line.strip
          end

          # If not found or string is invalid, nothing else to do
          next if !min_hash or !git_check_ref_format(min_hash)

          # Now find all commits between them
          IO.foreach("|git rev-list #{min_hash.shellescape}..#{max_hash.shellescape} --") do |line|
            hash = line.strip
            commits.push(hash) if !resolved_exclude or !resolved_exclude.include? hash
          end

          commits.push(min_hash) if !resolved_exclude or !resolved_exclude.include? min_hash
        else
          commits.push(max_hash) if !resolved_exclude or !resolved_exclude.include? max_hash
        end
      else
        logger.warn "Repository #{r.name} exists in table but not found on disk"
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
   Repository.find_each do |repo|
     if git_dir = repo.server_path
       @repositories[repo.name] = {git_dir: git_dir}
     end
   end

   @repositories
 end
end
