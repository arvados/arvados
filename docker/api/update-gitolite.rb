#!/usr/bin/env ruby

require 'rubygems'
require 'pp'
require 'arvados'
require 'tempfile'
require 'yaml'
require 'fileutils'

# This script does the actual gitolite config management on disk.
#
# Ward Vandewege <ward@curoverse.com>

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
  FileUtils.mkpath(dirname)
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

class UserSSHKeys
  include TrackCommitState

  def initialize(user_keys_map, key_dir)
    @user_keys_map = user_keys_map
    @key_dir = key_dir
    @installed = {}
  end

  def install(filename, pubkey)
    unless pubkey.nil?
      key_path = File.join(@key_dir, filename)
      ensure_in_git(key_path, pubkey)
    end
    @installed[filename] = true
  end

  def ensure_keys_for_user(user_uuid)
    return unless key_list = @user_keys_map.delete(user_uuid)
    key_list.map { |k| k[:public_key] }.compact.each_with_index do |pubkey, ii|
      # Handle putty-style ssh public keys
      pubkey.sub!(/^(Comment: "r[^\n]*\n)(.*)$/m,'ssh-rsa \2 \1')
      pubkey.sub!(/^(Comment: "d[^\n]*\n)(.*)$/m,'ssh-dss \2 \1')
      pubkey.gsub!(/\n/,'')
      pubkey.strip!
      install("#{user_uuid}@#{ii}.pub", pubkey)
    end
  end

  def installed?(filename)
    @installed[filename]
  end
end

class Repository
  include TrackCommitState

  @@aliases = {}

  def initialize(arv_repo, user_keys)
    @arv_repo = arv_repo
    @user_keys = user_keys
  end

  def self.ensure_system_config(conf_root)
    ensure_in_git(File.join(conf_root, "conf", "gitolite.conf"),
                  %Q{include "auto/*.conf"\ninclude "admin/*.conf"\n})
    ensure_in_git(File.join(conf_root, "arvadosaliases.pl"), alias_config)

    conf_path = File.join(conf_root, "conf", "admin", "arvados.conf")
    conf_file = %Q{
@arvados_git_user = arvados_git_user

repo gitolite-admin
     RW           = @arvados_git_user

}
    ensure_directory(File.dirname(conf_path), 0755)
    ensure_in_git(conf_path, conf_file)
  end

  def ensure_config(conf_root)
    if name and (File.exist?(auto_conf_path(conf_root, name)))
      # This gitolite installation knows the repository by name, rather than
      # UUID.  Leave it configured that way until a separate migration is run.
      basename = name
    else
      basename = uuid
      @@aliases[name] = uuid unless name.nil?
    end
    conf_file = "\nrepo #{basename}\n"
    @arv_repo[:user_permissions].sort.each do |user_uuid, perm|
      conf_file += "\t#{perm[:gitolite_permissions]}\t= #{user_uuid}\n"
      @user_keys.ensure_keys_for_user(user_uuid)
    end
    ensure_in_git(auto_conf_path(conf_root, basename), conf_file)
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

  ensure_directory(gitolite_keydir, 0700)
  user_ssh_keys = UserSSHKeys.new(permissions[:user_keys], gitolite_keydir)
  # Make sure the arvados_git_user key is installed
  user_ssh_keys.install('arvados_git_user.pub', gitolite_arvados_git_user_key)

  permissions[:repositories].each do |repo_record|
    repo = Repository.new(repo_record, user_ssh_keys)
    repo.ensure_config(gitolite_admin)
  end
  Repository.ensure_system_config(gitolite_admin)

  # Clean up public key files that should not be present
  Dir.chdir(gitolite_keydir)
  stale_keys = Dir.glob('*.pub').reject do |key_file|
    user_ssh_keys.installed?(key_file)
  end
  if stale_keys.any?
    stale_keys.each { |key_file| puts "Extra file #{key_file}" }
    system("git", "rm", "--quiet", *stale_keys)
  end

  if UserSSHKeys.changed? or Repository.changed? or stale_keys.any?
    message = "#{Time.now().to_s}: update from API"
    Dir.chdir(gitolite_admin)
    `git add --all`
    `git commit -m '#{message}'`
    `git push`
  end

rescue => bang
  puts "Error: " + bang.to_s
  puts bang.backtrace.join("\n")
  exit 1
end

