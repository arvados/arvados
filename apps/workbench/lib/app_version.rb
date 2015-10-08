# If you change this file, you'll probably also want to make the same
# changes in services/api/lib/app_version.rb.

class AppVersion
  def self.git(*args, &block)
    IO.popen(["git", "--git-dir", ".git"] + args, "r",
             chdir: Rails.root.join('../..'),
             err: "/dev/null",
             &block)
  end

  def self.forget
    @hash = nil
  end

  # Return abbrev commit hash for current code version: "abc1234", or
  # "abc1234-modified" if there are uncommitted changes. If present,
  # return contents of {root}/git-commit.version instead.
  def self.hash
    if (cached = Rails.configuration.source_version || @hash)
      return cached
    end

    # Read the version from our package's git-commit.version file, if available.
    begin
      @hash = IO.read(Rails.root.join("git-commit.version")).strip
    rescue Errno::ENOENT
    end

    if @hash.nil? or @hash.empty?
      begin
        local_modified = false
        git("status", "--porcelain") do |git_pipe|
          git_pipe.each_line do |_|
            STDERR.puts _
            local_modified = true
            # Continue reading the pipe so git doesn't get SIGPIPE.
          end
        end
        if $?.success?
          git("log", "-n1", "--format=%H") do |git_pipe|
            git_pipe.each_line do |line|
              @hash = line.chomp[0...8] + (local_modified ? '-modified' : '')
            end
          end
        end
      rescue SystemCallError
      end
    end

    @hash || "unknown"
  end
end
