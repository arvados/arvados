namespace :config do
  desc 'Ensure site configuration has all required settings'
  task check: :environment do
    $application_config.sort.each do |k, v|
      if ENV.has_key?('QUIET') then
        # Make sure we still check for the variable to exist
        eval("Rails.configuration.#{k}")
      else
        if /(password|secret)/.match(k) then
          # Make sure we still check for the variable to exist, but don't print the value
          eval("Rails.configuration.#{k}")
          $stderr.puts "%-32s %s" % [k, '*********']
        else
          $stderr.puts "%-32s %s" % [k, eval("Rails.configuration.#{k}")]
        end
      end
    end
    # default_trash_lifetime cannot be less than 24 hours
    if Rails.configuration.default_trash_lifetime < 86400 then
      raise "default_trash_lifetime is %d, must be at least 86400" % Rails.configuration.default_trash_lifetime
    end
  end
end
