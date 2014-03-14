namespace :config do
  desc 'Ensure site configuration has all required settings'
  task check: :environment do
    $application_config.sort.each do |k, v|
      $stderr.puts "%-32s %s" % [k, eval("Rails.configuration.#{k}")]
    end
  end
end
