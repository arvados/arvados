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

# Any _PW config settings represent a database password.  If it
# is blank, choose a password randomly.
config.each_key do |var|
  if var.end_with?('_PW') and (config[var].nil? or config[var].empty?)
    config[var] = rand(2**256).to_s(36)
  end
end

# ============================================================
# For each *.in file in the docker directories, substitute any
# @@variables@@ found in the file with the appropriate config
# variable.
# 
# TODO(twp): add the *.in files directory to the source tree, and
# when expanding them, add them to the "generated" directory with
# the same tree structure as in the original source. Then all
# the files can be added to the docker container with a single ADD.

Dir.glob('*/*.in') do |template_file|
  generated_dir = File.join(File.dirname(template_file), 'generated')
  Dir.mkdir(generated_dir) unless Dir.exists? generated_dir
  output_path = File.join(generated_dir, File.basename(template_file, '.in'))
  output = File.open(output_path, "w")
  File.open(template_file) do |input|
    input.each_line do |line|
      output.write(line.gsub(/@@(.*?)@@/) { |var| config[$1] || var })
    end
  end
  output.close
end
