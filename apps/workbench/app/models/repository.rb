class Repository < ArvadosBase
  def self.creatable?
    current_user and current_user.is_admin
  end
  def attributes_for_display
    super.reject { |x| x[0] == 'fetch_url' }
  end
  def editable_attributes
    if current_user.is_admin
      super
    else
      []
    end
  end

  def show commit_sha1
    refresh
    run_git 'show', commit_sha1
  end

  def cat_file commit_sha1, path
    refresh
    run_git 'cat-file', 'blob', commit_sha1 + ':' + path
  end

  def ls_tree_lr commit_sha1
    refresh
    run_git 'ls-tree', '-l', '-r', commit_sha1
  end

  # subtree returns a list of files under the given path at the
  # specified commit. Results are returned as an array of file nodes,
  # where each file node is an array [file mode, blob sha1, file size
  # in bytes, path relative to the given directory]. If the path is
  # not found, [] is returned.
  def ls_subtree commit, path
    path = path.chomp '/'
    subtree = []
    ls_tree_lr(commit).each_line do |line|
      mode, type, sha1, size, filepath = line.split
      next if type != 'blob'
      if filepath[0,path.length] == path and
          (path == '' or filepath[path.length] == '/')
        subtree << [mode.to_i(8), sha1, size.to_i,
                    filepath[path.length,filepath.length]]
      end
    end
    subtree
  end

  # git 2.1.4 does not use credential helpers reliably, see #5416
  def self.disable_repository_browsing?
    return false if Rails.configuration.use_git2_despite_bug_risk
    if @buggy_git_version.nil?
      @buggy_git_version = /git version 2/ =~ `git version`
    end
    @buggy_git_version
  end

  # http_fetch_url returns the first http:// or https:// url (if any)
  # in the api response's clone_urls attribute.
  def http_fetch_url
    clone_urls.andand.select { |u| /^http/ =~ u }.first
  end

  protected

  # refresh fetches the latest repository content into the local
  # cache. It is a no-op if it has already been run on this object:
  # this (pretty much) avoids doing more than one remote git operation
  # per Workbench request.
  def refresh
    run_git 'fetch', http_fetch_url, '+*:*' unless @fresh
    @fresh = true
  end

  # run_git sets up the ARVADOS_API_TOKEN environment variable,
  # creates a local git directory for this repository if necessary,
  # executes "git --git-dir localgitdir {args to run_git}", and
  # returns the output. It raises GitCommandError if git exits
  # non-zero.
  def run_git *gitcmd
    if not @workdir
      workdir = File.expand_path uuid+'.git', Rails.configuration.repository_cache
      if not File.exists? workdir
        FileUtils.mkdir_p Rails.configuration.repository_cache
        [['git', 'init', '--bare', workdir],
        ].each do |cmd|
          system *cmd
          raise GitCommandError.new($?.to_s) unless $?.exitstatus == 0
        end
      end
      @workdir = workdir
    end
    [['git', '--git-dir', @workdir, 'config', '--local',
      "credential.#{http_fetch_url}.username", 'none'],
     ['git', '--git-dir', @workdir, 'config', '--local',
      "credential.#{http_fetch_url}.helper",
      '!token(){ echo password="$ARVADOS_API_TOKEN"; }; token'],
     ['git', '--git-dir', @workdir, 'config', '--local',
           'http.sslVerify',
           Rails.configuration.arvados_insecure_https ? 'false' : 'true'],
     ].each do |cmd|
      system *cmd
      raise GitCommandError.new($?.to_s) unless $?.exitstatus == 0
    end
    env = {}.
      merge(ENV).
      merge('ARVADOS_API_TOKEN' => Thread.current[:arvados_api_token])
    cmd = ['git', '--git-dir', @workdir] + gitcmd
    io = IO.popen(env, cmd, err: [:child, :out])
    output = io.read
    io.close
    # "If [io] is opened by IO.popen, close sets $?." --ruby 2.2.1 docs
    unless $?.exitstatus == 0
      raise GitCommandError.new("`git #{gitcmd.join ' '}` #{$?}: #{output}")
    end
    output
  end

  class GitCommandError < StandardError
  end
end
