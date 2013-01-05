set :application, "orvos-server"
set :domain,      "controller.van"
set :deploy_to,   "/var/www/orvos.clinicalfuture.com"
role :web, "controller.van"
role :app, "controller.van"
role :db, "controller.van", :primary=>true
set :scm,         :git
set :repository,  "git@git.clinicalfuture.com:orvos-server.git"
set :rails_env,   "production"
set :config_files, ['database.yml']
set :git_enable_submodules, true

ssh_options[:forward_agent] = true
ssh_options[:user] = 'root'

desc "Clean up old releases"
set :keep_releases, 5
before("deploy:cleanup") { set :use_sudo, false }

after "deploy:create_symlink", "deploy:copy_files", :roles => :app
# after "deploy:create_symlink", "deploy:bundle_install", :roles => :app
after "deploy:update", "deploy:migrate", :roles => :db
after :deploy, 'deploy:cleanup', :roles => :app

namespace :deploy do
  desc "Put a few files in place, ensure correct file ownership"
  task :copy_files, :roles => :app,  :except => { :no_release => true } do
    # Copy a few files/make a few symlinks
    run "cp /home/passenger/orvos-server/database.yml #{current_path}/config/database.yml"
    run "cp /home/passenger/orvos-server/secret_token.rb #{current_path}/config/initializers/secret_token.rb"
    run "cp /home/passenger/orvos-server/production.rb #{current_path}/config/environments/production.rb"
    # Ensure correct ownership of a few files
    run "chown www-data:www-data #{current_path}/config/environment.rb"
    run "chown www-data:www-data #{current_path}/config.ru"
    # This is for the drb server
    run "touch #{current_path}/Gemfile.lock"
    run "chown www-data:www-data #{current_path}/Gemfile.lock"
    # Keep track of the git commit used for this deploy
    # This is used by the lib/add_debug_info.rb middleware, which injects it in the
    # environment.
    run "cd #{current_path}; /usr/bin/git rev-parse HEAD > #{current_path}/git-commit.version"
  end

  # desc "Install new gems if necessary"
  # task :bundle_install, :roles => :app,  :except => { :no_release => true } do
  #   run "cd #{current_path} && bundle install --local"
  # end

  desc "Restarting mod_rails with restart.txt"
  task :restart, :roles => :app, :except => { :no_release => true } do
    # Tell passenger to restart.
    run "touch #{current_path}/tmp/restart.txt"
    # Tell DRB to restart.
    run "/usr/sbin/monit restart mypg_server.rb"
  end 
  [:start, :stop].each do |t| 
    desc "#{t} task is a no-op with mod_rails"
    task t, :roles => :app do ; end 
  end 
end
