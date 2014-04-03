#! /usr/bin/env ruby

require 'yaml'
require 'fileutils'

abort 'Error: Ruby >= 1.9.3 required.' if RUBY_VERSION < '1.9.3'

# Initialize config settings from config.yml
config = YAML.load_file('config.yml')

# ============================================================
# Add dynamically chosen config settings. These settings should
# be suitable for any installation.

# Any _PW/_SECRET config settings represent passwords/secrets. If they
# are blank, choose a password randomly.
config.each_key do |var|
  if (var.end_with?('_PW') || var.end_with?('_SECRET')) && (config[var].nil? || config[var].empty?)
    config[var] = rand(2**256).to_s(36)
  end
end

# ============================================================
# For each *.in file in the docker directories, substitute any
# @@variables@@ found in the file with the appropriate config
# variable. Support up to 10 levels of nesting.
#
# TODO(twp): add the *.in files directory to the source tree, and
# when expanding them, add them to the "generated" directory with
# the same tree structure as in the original source. Then all
# the files can be added to the docker container with a single ADD.

Dir.glob('*/generated/*') do |stale_file|
  File.delete(stale_file)
end

File.umask(022)
Dir.glob('*/*.in') do |template_file|
  generated_dir = File.join(File.dirname(template_file), 'generated')
  Dir.mkdir(generated_dir) unless Dir.exists? generated_dir
  output_path = File.join(generated_dir, File.basename(template_file, '.in'))
  File.open(output_path, "w") do |output|
    File.open(template_file) do |input|
      input.each_line do |line|

        # This count is used to short-circuit potential
        # infinite loops of variable substitution.
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
  end
end

# Copy the ssh public key file to base/generated (if a path is given)
generated_dir = File.join('base/generated')
Dir.mkdir(generated_dir) unless Dir.exists? generated_dir
if (config['PUBLIC_KEY_PATH'] != nil and
    File.readable? config['PUBLIC_KEY_PATH'])
  FileUtils.cp(config['PUBLIC_KEY_PATH'],
               File.join(generated_dir, 'id_rsa.pub'))
end
