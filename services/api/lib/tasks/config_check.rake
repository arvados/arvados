namespace :config do
  desc 'Ensure site configuration has all required settings'
  task check: :environment do
    $application_config.sort.each do |k, v|
      if /(password|secret)/.match(k) then
        $stderr.puts "%-32s %s" % [k, '*********']
      else
        $stderr.puts "%-32s %s" % [k, eval("Rails.configuration.#{k}")]
      end
    end
  end
end
