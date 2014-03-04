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
  opt :force, <<-eos
Continue even if sanity checks raise flags: the given user is already \
active, the given repository already exists, etc.
  eos
  opt :n, 'Do not change anything, just probe'
end

log.level = (ENV['DEBUG'] || opts.debug) ? Logger::DEBUG : Logger::WARN
    
if ARGV.count != 3
  Trollop::die "required arguments are missing"
end
user_arg, user_repo_name, vm_uuid = ARGV

require 'arvados'
arv = Arvados.new(api_version: 'v1')

# Look up the given user by uuid or, failing that, email address.
user = begin
         arv.user.get(uuid: user_arg)
       rescue Arvados::TransactionFailedError
         found = arv.user.list(where: {email: ARGV[0]})[:items]
         if found.count == 0 and opts.create
           if !opts.force and !user_arg.match(/\w\@\w+\.\w+/)
             abort "About to create new user, but #{user_arg.inspect} " +
               "does not look like an email address. Stop."
           end
           if opts.n
             log.info "-n flag given. Stop before creating new user record."
             exit 0
           end
           new_user = arv.user.create(user: {email: user_arg})
           log.info { "created user: " + new_user[:uuid] }
           login_perm_props = {identity_url_prefix: opts.openid_prefix }
           oid_login_perm = arv.link.create(link: {
                                              link_class: 'permission',
                                              name: 'can_login',
                                              tail_kind: 'email',
                                              tail_uuid: user_arg,
                                              head_kind: 'arvados#user',
                                              head_uuid: new_user[:uuid],
                                              properties: login_perm_props
                                            })
           log.info { "openid login permission: " + oid_login_perm[:uuid] }
           found = [new_user]
         end
         if found.count != 1
           abort "Found #{found.count} users " +
             "with uuid or email #{user_arg.inspect}. Stop."
         end
         found.first
       end
log.info { "user uuid: " + user[:uuid] }

# Look up the given virtual machine just to make sure it really exists.
begin
  vm = arv.virtual_machine.get(uuid: vm_uuid)
rescue
  abort "Could not look up virtual machine with uuid #{vm_uuid.inspect}. Stop."
end
log.info { "vm uuid: " + vm[:uuid] }

# Look up the "All users" group (we expect uuid *-*-fffffffffffffff).
group = arv.group.list(where: {name: 'All users'})[:items].select do |g|
  g[:uuid].match /-f+$/
end.first
if not group
  abort "Could not look up the 'All users' group with uuid '*-*-fffffffffffffff'. Stop."
end
log.info { "\"All users\" group uuid: " + group[:uuid] }

# Look for signs the user has already been activated / set up.

if user[:is_active]
  log.warn "User's is_active flag is already set."
  need_force = true
end

# Look for existing repository access (perhaps using a different
# repository/user name).
repo_perms = arv.link.list(where: {
                             tail_uuid: user[:uuid],
                             head_kind: 'arvados#repository',
                             link_class: 'permission',
                             name: 'can_write'})[:items]
if [] != repo_perms
  log.warn "User already has repository access " +
    repo_perms.collect { |p| p[:uuid] }.inspect + "."
  need_force = true
end

# Check for an existing repository with the same name we're about to
# use.
repo = arv.repository.list(where: {name: user_repo_name})[:items].first
if repo
  log.warn "Repository already exists with name #{user_repo_name.inspect}: " +
    "#{repo[:uuid]}"
  need_force = true
end

if opts.n
  log.info "-n flag given. Done."
  exit 0
end

if need_force and not opts.force
  abort "This does not seem to be a new user[name], and -f was not given. Stop."
end

# Everything seems to be in order. Create a repository (if needed) and
# add permissions.

repo ||= arv.repository.create(repository: {name: user_repo_name})
log.info { "repo uuid: " + repo[:uuid] }

repo_perm = arv.link.create(link: {
                              tail_kind: 'arvados#user',
                              tail_uuid: user[:uuid],
                              head_kind: 'arvados#repository',
                              head_uuid: repo[:uuid],
                              link_class: 'permission',
                              name: 'can_write'})
log.info { "repo permission: " + repo_perm[:uuid] }

login_perm = arv.link.create(link: {
                               tail_kind: 'arvados#user',
                               tail_uuid: user[:uuid],
                               head_kind: 'arvados#virtualMachine',
                               head_uuid: vm[:uuid],
                               link_class: 'permission',
                               name: 'can_login',
                               properties: {username: user_repo_name}})
log.info { "login permission: " + login_perm[:uuid] }

group_perm = arv.link.create(link: {
                               tail_kind: 'arvados#user',
                               tail_uuid: user[:uuid],
                               head_kind: 'arvados#group',
                               head_uuid: group[:uuid],
                               link_class: 'permission',
                               name: 'can_read'})
log.info { "group permission: " + group_perm[:uuid] }
