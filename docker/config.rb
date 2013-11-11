#! /usr/bin/env ruby

require 'yaml'

# Initialize config settings from config.yml
config = YAML.load_file('config.yml')

# ============================================================
# Add dynamically chosen config settings. These settings should
# be suitable for any installation.

# The APP_SECRET the application uses (with OMNIAUTH_APP_ID) to
# authenticate itself to Omniauth. By default this is generated
# randomly when the application is built; you can instead
# substitute a hardcoded string.
config['OMNIAUTH_APP_SECRET'] = rand(2**512).to_s(36)

# The secret token in services/api/config/initializers/secret_token.rb.
config['API_SECRET'] = rand(2**256).to_s(36)
config['WORKER_SECRET'] = rand(2**256).to_s(36)

# Any _PW config settings represent a database password.  If it
# is blank, choose a password randomly.
config.each_key do |var|
  if var.end_with?('_PW') && (config[var].nil? || config[var].empty?)
    config[var] = rand(2**256).to_s(36)
  end
end

# ============================================================
# For each *.in file in the docker directories, substitute any
# @@variables@@ found in the file with the appropriate config
# variable. Support up to 10 levels of nesting.

Dir.glob('*/*.in') do |template_file|
  output_path = template_file.sub(/\.in$/, '')
  output = File.open(output_path, 'w')
  File.open(template_file) do |input|
    input.each_line do |line|

      @count = 0
      while @count < 10
        @out = line.gsub!(/@@(.*?)@@/) do |var|
          if config.key?(Regexp.last_match[1])
            config[Regexp.last_match[1]]
          else
            var.gsub!(/@@/, '@_NOT_FOUND_@')
          end
        end
        break if @out.nil?
        @count += 1
      end

      output.write(line)
    end
  end
  output.close
end
