set :application, "vcffarm"
set :domain,      "orvos-app-0.ant"
set :deploy_to,   "/var/www/orvos-explorer.ant.freelogy.org"
role :web, "orvos-app-0.ant"
role :app, "orvos-app-0.ant"
set :scm,         :git
set :repository,  "git@git.clinicalfuture.com:vcffarm.git"
set :rails_env,   "production"
set :git_enable_submodules, true
set :rvm_ruby_string, '1.9.3'
require "rvm/capistrano"
load "deploy/assets"
default_run_options[:shell] = '/bin/bash --login'

ssh_options[:forward_agent] = true
ssh_options[:user] = 'root'

desc "Clean up old releases"
set :keep_releases, 5
before("deploy:cleanup") { set :use_sudo, false }

before "deploy:assets:precompile", "deploy:copy_files", :roles => :app
after "deploy:copy_files", "deploy:bundle_install", :roles => :app
after :deploy, 'deploy:cleanup', :roles => :app

namespace :deploy do
  desc "Put a few files in place, ensure correct file ownership"
  task :copy_files, :roles => :app,  :except => { :no_release => true } do
    # Copy a few files/make a few symlinks
    run "cp /home/passenger/explorer/secret_token.rb #{release_path}/config/initializers/secret_token.rb"
    run "cp /home/passenger/explorer/production.rb #{release_path}/config/environments/production.rb"
    # Ensure correct ownership of a few files
    run "chown www-data:www-data #{release_path}/config/environment.rb"
    run "chown www-data:www-data #{release_path}/config.ru"
    run "touch #{release_path}/db/production.sqlite3"
    run "chown www-data:www-data #{release_path}/db/production.sqlite3"
    run "chown root:www-data #{release_path}/db"
    run "chmod g+w,+t #{release_path}/db"
    # Keep track of the git commit used for this deploy
    # This is used by the lib/add_debug_info.rb middleware, which injects it in the
    # environment.
    run "cd #{release_path}; /usr/bin/git rev-parse HEAD > #{release_path}/git-commit.version"

    # make sure to symlink the vendor bundle. Cf. http://gembundler.com/rationale.html
    run "cd #{release_path}; ln -s #{shared_path}/vendor_bundle #{release_path}/vendor/bundle"
  end

  # desc "Install new gems if necessary"
  task :bundle_install, :roles => :app,  :except => { :no_release => true } do
    run "cd #{release_path} && bundle install --deployment"
  end

  desc "Restarting mod_rails with restart.txt"
  task :restart, :roles => :app, :except => { :no_release => true } do
    # Tell passenger to restart.
    run "touch #{release_path}/tmp/restart.txt"
  end

end
