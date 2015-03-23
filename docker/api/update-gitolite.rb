#!/usr/bin/env ruby

require 'rubygems'
require 'pp'
require 'arvados'
require 'active_support/all'
require 'yaml'

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
gitolite_tmp = cp_config['gitolite_tmp']

gitolite_admin = File.join(File.expand_path(File.dirname(__FILE__)) + '/' + gitolite_tmp + '/gitolite-admin')

ENV['ARVADOS_API_HOST'] = cp_config['arvados_api_host']
ENV['ARVADOS_API_TOKEN'] = cp_config['arvados_api_token']
if cp_config['arvados_api_host_insecure']
  ENV['ARVADOS_API_HOST_INSECURE'] = 'true'
else
  ENV.delete('ARVADOS_API_HOST_INSECURE')
end

keys = ''

seen = Hash.new

def ensure_repo(name,permissions,user_keys,gitolite_admin)
  tmp = ''
  # Just in case...
  name.gsub!(/[^a-z0-9]/i,'')

  keys = Hash.new()

  user_keys.each do |uuid,p|
    p.each do |k|
      next if k[:public_key].nil?
      keys[uuid] = Array.new() if not keys.key?(uuid)

      key = k[:public_key]
      # Handle putty-style ssh public keys
      key.sub!(/^(Comment: "r[^\n]*\n)(.*)$/m,'ssh-rsa \2 \1')
      key.sub!(/^(Comment: "d[^\n]*\n)(.*)$/m,'ssh-dss \2 \1')
      key.gsub!(/\n/,'')
      key.strip

      keys[uuid].push(key)
    end
  end

  cf = gitolite_admin + '/conf/auto/' + name + '.conf'

  conf = "\nrepo #{name}\n"

  commit = false

  seen = {}
  permissions.sort.each do |uuid,v|
    conf += "\t#{v[:gitolite_permissions]}\t= #{uuid.to_s}\n"

    count = 0
    keys.include?(uuid) and keys[uuid].each do |v|
      kf = gitolite_admin + '/keydir/arvados/' + uuid.to_s + "@#{count}.pub"
      seen[kf] = true
      if !File.exists?(kf) or IO::read(kf) != v then
        commit = true
        f = File.new(kf + ".tmp",'w')
        f.write(v)
        f.close()
        # File.rename will overwrite the destination file if it exists
        File.rename(kf + ".tmp",kf);
      end
      count += 1
    end
  end

  if !File.exists?(cf) or IO::read(cf) != conf then
    commit = true
    f = File.new(cf + ".tmp",'w')
    f.write(conf)
    f.close()
    # this is about as atomic as we can make the replacement of the file...
    File.unlink(cf) if File.exists?(cf)
    File.rename(cf + ".tmp",cf);
  end

  return commit,seen
end

begin

  pwd = Dir.pwd
  # Get our local gitolite-admin repo up to snuff
  if not File.exists?(File.dirname(__FILE__) + '/' + gitolite_tmp) then
    Dir.mkdir(File.join(File.dirname(__FILE__) + '/' + gitolite_tmp), 0700)
  end
  if not File.exists?(gitolite_admin) then
    Dir.chdir(File.join(File.dirname(__FILE__) + '/' + gitolite_tmp))
    `git clone #{gitolite_url}`
  else
    Dir.chdir(gitolite_admin)
    `git pull`
  end
  Dir.chdir(pwd)

  arv = Arvados.new( { :suppress_ssl_warnings => false } )

  permissions = arv.repository.get_all_permissions

  repos = permissions[:repositories]
  user_keys = permissions[:user_keys]

  @commit = false

  @seen = {}

  repos.each do |r|
    next if r[:name].nil?
    (@c,@s) = ensure_repo(r[:name],r[:user_permissions],user_keys,gitolite_admin)
    @seen.merge!(@s)
    @commit = true if @c
  end

  # Clean up public key files that should not be present
  Dir.glob(gitolite_admin + '/keydir/arvados/*.pub') do |key_file|
    next if key_file =~ /arvados_git_user.pub$/
    next if @seen.has_key?(key_file)
    puts "Extra file #{key_file}"
    @commit = true
    Dir.chdir(gitolite_admin)
    key_file.gsub!(/^#{gitolite_admin}\//,'')
    `git rm #{key_file}`
  end

  if @commit then
    message = "#{Time.now().to_s}: update from API"
    Dir.chdir(gitolite_admin)
    `git add --all`
    `git commit -m '#{message}'`
    `git push`
  end

rescue Exception => bang
  puts "Error: " + bang.to_s
  puts bang.backtrace.join("\n")
  exit 1
end

