class Commit < ActiveRecord::Base
  extend CurrentApiClient

  class GitError < StandardError
    def http_status
      422
    end
  end

  def self.git_check_ref_format(e)
    if !e or e.empty? or e[0] == '-' or e[0] == '$'
      # definitely not valid
      false
    else
      `git check-ref-format --allow-onelevel #{e.shellescape}`
      $?.success?
    end
  end

  # Return an array of commits (each a 40-char sha1) satisfying the
  # given criteria.
  #
  # Return [] if the revisions given in minimum/maximum are invalid or
  # don't exist in the given repository.
  #
  # Raise ArgumentError if the given repository is invalid, does not
  # exist, or cannot be read for any reason. (Any transient error that
  # prevents commit ranges from resolving must raise rather than
  # returning an empty array.)
  #
  # repository can be the name of a locally hosted repository or a git
  # URL (see git-fetch(1)). Currently http, https, and git schemes are
  # supported.
  def self.find_commit_range repository, minimum, maximum, exclude
    if minimum and minimum.empty?
      minimum = nil
    end

    if minimum and !git_check_ref_format(minimum)
      logger.warn "find_commit_range called with invalid minimum revision: '#{minimum}'"
      return []
    end

    if maximum and !git_check_ref_format(maximum)
      logger.warn "find_commit_range called with invalid maximum revision: '#{maximum}'"
      return []
    end

    if !maximum
      maximum = "HEAD"
    end

    gitdir, is_remote = git_dir_for repository
    fetch_remote_repository gitdir, repository if is_remote
    ENV['GIT_DIR'] = gitdir

    commits = []

    # Get the commit hash for the upper bound
    max_hash = nil
    IO.foreach("|git rev-list --max-count=1 #{maximum.shellescape} --") do |line|
      max_hash = line.strip
    end

    # If not found or string is invalid, nothing else to do
    return [] if !max_hash or !git_check_ref_format(max_hash)

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
          return []
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
      return [] if !min_hash or !git_check_ref_format(min_hash)

      # Now find all commits between them
      IO.foreach("|git rev-list #{min_hash.shellescape}..#{max_hash.shellescape} --") do |line|
        hash = line.strip
        commits.push(hash) if !resolved_exclude or !resolved_exclude.include? hash
      end

      commits.push(min_hash) if !resolved_exclude or !resolved_exclude.include? min_hash
    else
      commits.push(max_hash) if !resolved_exclude or !resolved_exclude.include? max_hash
    end

    commits
  end

  # Given a repository (url, or name of hosted repo) and commit sha1,
  # copy the commit into the internal git repo and tag it with the
  # given tag (typically a job UUID).
  #
  # The repo can be a remote url, but in this case sha1 must already
  # be present in our local cache for that repo: e.g., sha1 was just
  # returned by find_commit_range.
  def self.tag_in_internal_repository repo_name, sha1, tag
    unless git_check_ref_format tag
      raise ArgumentError.new "invalid tag #{tag}"
    end
    unless /^[0-9a-f]{40}$/ =~ sha1
      raise ArgumentError.new "invalid sha1 #{sha1}"
    end
    src_gitdir, _ = git_dir_for repo_name
    dst_gitdir = Rails.configuration.git_internal_dir
    must_pipe("echo #{sha1.shellescape}",
              "git --git-dir #{src_gitdir.shellescape} pack-objects -q --revs --stdout",
              "git --git-dir #{dst_gitdir.shellescape} unpack-objects -q")
    must_git(dst_gitdir,
             "tag --force #{tag.shellescape} #{sha1.shellescape}")
  end

  protected

  def self.remote_url? repo_name
    /^(https?|git):\/\// =~ repo_name
  end

  # Return [local_git_dir, is_remote]. If is_remote, caller must use
  # fetch_remote_repository to ensure content is up-to-date.
  #
  # Raises an exception if the latest content could not be fetched for
  # any reason.
  def self.git_dir_for repo_name
    if remote_url? repo_name
      return [cache_dir_for(repo_name), true]
    end
    repos = Repository.readable_by(current_user).where(name: repo_name)
    if repos.count == 0
      raise ArgumentError.new "Repository not found: '#{repo_name}'"
    elsif repos.count > 1
      logger.error "Multiple repositories with name=='#{repo_name}'!"
      raise ArgumentError.new "Name conflict"
    else
      return [repos.first.server_path, false]
    end
  end

  def self.cache_dir_for git_url
    File.join(cache_dir_base, Digest::SHA1.hexdigest(git_url) + ".git").to_s
  end

  def self.cache_dir_base
    Rails.root.join 'tmp', 'git'
  end

  def self.fetch_remote_repository gitdir, git_url
    # Caller decides which protocols are worth using. This is just a
    # safety check to ensure we never use urls like "--flag" or wander
    # into git's hardlink features by using bare "/path/foo" instead
    # of "file:///path/foo".
    unless /^[a-z]+:\/\// =~ git_url
      raise ArgumentError.new "invalid git url #{git_url}"
    end
    begin
      must_git gitdir, "branch"
    rescue GitError => e
      raise unless /Not a git repository/ =~ e.to_s
      # OK, this just means we need to create a blank cache repository
      # before fetching.
      FileUtils.mkdir_p gitdir
      must_git gitdir, "init"
    end
    must_git(gitdir,
             "fetch --no-progress --tags --prune --force --update-head-ok #{git_url.shellescape} 'refs/heads/*:refs/heads/*'")
  end

  def self.must_git gitdir, *cmds
    # Clear token in case a git helper tries to use it as a password.
    orig_token = ENV['ARVADOS_API_TOKEN']
    ENV['ARVADOS_API_TOKEN'] = ''
    begin
      git = "git --git-dir #{gitdir.shellescape}"
      cmds.each do |cmd|
        must_pipe git+" "+cmd
      end
    ensure
      ENV['ARVADOS_API_TOKEN'] = orig_token
    end
  end

  def self.must_pipe *cmds
    cmd = cmds.join(" 2>&1 |") + " 2>&1"
    out = IO.read("| </dev/null #{cmd}")
    if not $?.success?
      raise GitError.new "#{cmd}: #{$?}: #{out}"
    end
  end
end
