#!/usr/bin/env ruby
#
# Prior to April 2015, Arvados Gitolite integration stored repositories by
# name.  To improve user repository management, we switched to storing
# repositories by UUID, and aliasing them to names.  This makes it easy to
# have rich name hierarchies, and allow users to rename repositories.
#
# This script will migrate a name-based Gitolite configuration to a UUID-based
# one.  To use it:
#
# 1. Change the value of REPOS_DIR below, if needed.
# 2. Install this script in the same directory as `update-gitolite.rb`.
# 3. Ensure that no *other* users can access Gitolite: edit gitolite's
#    authorized_keys file so it only contains the arvados_git_user key,
#    and disable the update-gitolite cron job.
# 4. Run this script: `ruby migrate-gitolite-to-uuid-storage.rb production`.
# 5. Undo step 3.

require 'rubygems'
require 'pp'
require 'arvados'
require 'tempfile'
require 'yaml'

REPOS_DIR = "/var/lib/gitolite/repositories"

# Default is development
production = ARGV[0] == "production"

ENV["RAILS_ENV"] = "development"
ENV["RAILS_ENV"] = "production" if production

DEBUG = 1

# load and merge in the environment-specific application config info
# if present, overriding base config parameters as specified
path = File.dirname(__FILE__) + '/config/arvados-clients.yml'
if File.exists?(path) then
  cp_config = YAML.load_file(path)[ENV['RAILS_ENV']]
else
  puts "Please create a\n " + File.dirname(__FILE__) + "/config/arvados-clients.yml\n file"
  exit 1
end

gitolite_url = cp_config['gitolite_url']
gitolite_arvados_git_user_key = cp_config['gitolite_arvados_git_user_key']

gitolite_tmpdir = File.join(File.absolute_path(File.dirname(__FILE__)),
                            cp_config['gitolite_tmp'])
gitolite_admin = File.join(gitolite_tmpdir, 'gitolite-admin')
gitolite_keydir = File.join(gitolite_admin, 'keydir', 'arvados')

ENV['ARVADOS_API_HOST'] = cp_config['arvados_api_host']
ENV['ARVADOS_API_TOKEN'] = cp_config['arvados_api_token']
if cp_config['arvados_api_host_insecure']
  ENV['ARVADOS_API_HOST_INSECURE'] = 'true'
else
  ENV.delete('ARVADOS_API_HOST_INSECURE')
end

def ensure_directory(path, mode)
  begin
    Dir.mkdir(path, mode)
  rescue Errno::EEXIST
  end
end

def replace_file(path, contents)
  unlink_now = true
  dirname, basename = File.split(path)
  new_file = Tempfile.new([basename, ".tmp"], dirname)
  begin
    new_file.write(contents)
    new_file.flush
    File.rename(new_file, path)
    unlink_now = false
  ensure
    new_file.close(unlink_now)
  end
end

def file_has_contents?(path, contents)
  begin
    IO.read(path) == contents
  rescue Errno::ENOENT
    false
  end
end

module TrackCommitState
  module ClassMethods
    # Note that all classes that include TrackCommitState will have
    # @@need_commit = true if any of them set it.  Since this flag reports
    # a boolean state of the underlying git repository, that's OK in the
    # current implementation.
    @@need_commit = false

    def changed?
      @@need_commit
    end

    def ensure_in_git(path, contents)
      unless file_has_contents?(path, contents)
        replace_file(path, contents)
        system("git", "add", path)
        @@need_commit = true
      end
    end
  end

  def ensure_in_git(path, contents)
    self.class.ensure_in_git(path, contents)
  end

  def self.included(base)
    base.extend(ClassMethods)
  end
end

class Repository
  include TrackCommitState

  @@aliases = {}

  def initialize(arv_repo)
    @arv_repo = arv_repo
  end

  def self.ensure_system_config(conf_root)
    ensure_in_git(File.join(conf_root, "arvadosaliases.pl"), alias_config)
  end

  def self.rename_repos(repos_root)
    @@aliases.each_pair do |uuid, name|
      begin
        File.rename(File.join(repos_root, "#{name}.git/"),
                    File.join(repos_root, "#{uuid}.git"))
      rescue Errno::ENOENT
      end
    end
  end

  def ensure_config(conf_root)
    return if name.nil?
    @@aliases[uuid] = name
    name_conf_path = auto_conf_path(conf_root, name)
    return unless File.exist?(name_conf_path)
    conf_file = IO.read(name_conf_path)
    conf_file.gsub!(/^repo #{Regexp.escape(name)}$/m, "repo #{uuid}")
    ensure_in_git(auto_conf_path(conf_root, uuid), conf_file)
    File.unlink(name_conf_path)
    system("git", "rm", "--quiet", name_conf_path)
  end

  private

  def auto_conf_path(conf_root, basename)
    File.join(conf_root, "conf", "auto", "#{basename}.conf")
  end

  def uuid
    @arv_repo[:uuid]
  end

  def name
    if @arv_repo[:name].nil?
      nil
    else
      @clean_name ||=
        @arv_repo[:name].sub(/^[^A-Za-z]+/, "").gsub(/[^\w\.\/]/, "")
    end
  end

  def self.alias_config
    conf_s = "{\n"
    @@aliases.sort.each do |(repo_name, repo_uuid)|
      conf_s += "\t'#{repo_name}' \t=> '#{repo_uuid}',\n"
    end
    conf_s += "};\n"
    conf_s
  end
end

begin
  # Get our local gitolite-admin repo up to snuff
  if not File.exists?(gitolite_admin) then
    ensure_directory(gitolite_tmpdir, 0700)
    Dir.chdir(gitolite_tmpdir)
    `git clone #{gitolite_url}`
    Dir.chdir(gitolite_admin)
  else
    Dir.chdir(gitolite_admin)
    `git pull`
  end

  arv = Arvados.new
  permissions = arv.repository.get_all_permissions

  permissions[:repositories].each do |repo_record|
    repo = Repository.new(repo_record)
    repo.ensure_config(gitolite_admin)
  end
  Repository.ensure_system_config(gitolite_admin)

  message = "#{Time.now().to_s}: migrate to storing repositories by UUID"
  Dir.chdir(gitolite_admin)
  `git add --all`
  `git commit -m '#{message}'`
  Repository.rename_repos(REPOS_DIR)
  `git push`

rescue => bang
  puts "Error: " + bang.to_s
  puts bang.backtrace.join("\n")
  exit 1
end

