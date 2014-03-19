#!/usr/bin/env ruby

abort 'Error: Ruby >= 1.9.3 required.' if RUBY_VERSION < '1.9.3'

require 'logger'
require 'trollop'
log = Logger.new STDERR
log.progname = $0.split('/').last

opts = Trollop::options do
  banner ''
  banner "Usage: #{log.progname} " +
    "{user_uuid_or_email} {user_and_repo_name} {vm_uuid}"
  banner ''
  opt :debug, <<-eos
Show debug messages.
  eos
  opt :create, <<-eos
Create a new user with the given email address if an existing user \
is not found.
  eos
  opt :openid_prefix, <<-eos, default: 'https://www.google.com/accounts/o8/id'
If creating a new user record, require authentication from an OpenID \
with this OpenID prefix *and* a matching email address in order to \
claim the account.
  eos
end

log.level = (ENV['DEBUG'] || opts.debug) ? Logger::DEBUG : Logger::WARN
    
if ARGV.count != 3
  Trollop::die "required arguments are missing"
end
user_arg, user_repo_name, vm_uuid = ARGV

require 'arvados'
arv = Arvados.new(api_version: 'v1')

begin
  new_user = arv.user.create(user_param: user_arg, repo_name: user_repo_name, vm_uuid: vm_uuid, openid_prefix: opts.openid_prefix, user: {})
  log.warn new_user
rescue Exception => e #Arvados::TransactionFailedError
  log.warn e.message
end
