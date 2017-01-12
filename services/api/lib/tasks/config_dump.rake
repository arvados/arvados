namespace :config do
  desc 'Show site configuration'
  task dump: :environment do
    puts $application_config.to_yaml
  end
end
