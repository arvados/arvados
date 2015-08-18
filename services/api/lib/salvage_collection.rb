module SalvageCollection
  # Take two input parameters: a collection uuid and reason
  # Get "src_collection" with the given uuid
  # Create a new collection with:
  #   src_collection.manifest_text as "invalid_manifest_text.txt"
  #   Locators from src_collection.manifest_text as "salvaged_data"
  # Update src_collection:
  #   Set src_collection.manifest_text to: ""
  #   Append to src_collection.name: " (reason; salvaged data at new_collection.uuid)"
  #   Set portable_data_hash to "d41d8cd98f00b204e9800998ecf8427e+0"

  require File.dirname(__FILE__) + '/../config/environment'
  require 'arvados/keep'
  include ApplicationHelper
  require 'tempfile'
  require 'shellwords'

  def self.salvage_collection_arv_put temp_file
    %x(arv-put --as-stream --use-filename invalid_manifest_text.txt #{Shellwords::shellescape(temp_file.path)})
  end

  def self.salvage_collection uuid, reason='salvaged - see #6277, #6859'
    act_as_system_user do
      if !ENV['ARVADOS_API_TOKEN'].present? or !ENV['ARVADOS_API_HOST'].present?
        $stderr.puts "Please set your admin user credentials as ARVADOS environment variables."
        # exit with a code outside the range of special exit codes; http://tldp.org/LDP/abs/html/exitcodes.html
        exit 200
      end

      if !uuid.present?
        $stderr.puts "Required uuid argument is missing."
        return false
      end

      src_collection = Collection.find_by_uuid uuid
      if !src_collection
        $stderr.puts "No collection found for #{uuid}. Returning."
        return false
      end

      begin
        src_manifest = src_collection.manifest_text || ''

        # Get all the locators from the original manifest
        locators = []
        src_manifest.each_line do |line|
          line.split(' ').each do |word|
            if match = Keep::Locator::LOCATOR_REGEXP.match(word)
              word = word.split('+')[0..1].join('+')  # get rid of any hints
              locators << word if !word.start_with?('00000000000000000000000000000000')
            end
          end
        end
        locators << 'd41d8cd98f00b204e9800998ecf8427e+0' if !locators.any?

        # create new collection using 'arv-put' with original manifest_text as the data
        temp_file = Tempfile.new('temp')
        temp_file.write(src_manifest)
        temp_file.close

        new_manifest = salvage_collection_arv_put temp_file

        temp_file.unlink

        if !new_manifest.present?
          $stderr.puts "arv-put --as-stream failed for #{uuid}"
          return false
        end

        words = []
        new_manifest.split(' ').each do |word|
          if match = Keep::Locator::LOCATOR_REGEXP.match(word)
            word = word.split('+')[0..1].join('+')  # get rid of any hints
            words << word
          else
            words << word
          end
        end

        new_manifest = words.join(' ') + "\n"
        new_collection = Collection.new

        total_size = 0
        locators.each do |locator|
          total_size += locator.split('+')[1].to_i
        end
        new_manifest += (". #{locators.join(' ')} 0:#{total_size}:salvaged_data\n")

        new_collection.name = "salvaged from #{src_collection.uuid}, #{src_collection.portable_data_hash}"
        new_collection.manifest_text = new_manifest
        new_collection.portable_data_hash = Digest::MD5.hexdigest(new_collection.manifest_text)

        created = new_collection.save!
        raise "New collection creation failed." if !created

        $stderr.puts "Salvaged manifest and data for #{uuid} are in #{new_collection.uuid}."
        puts "Created new collection #{new_collection.uuid}"
      rescue => error
        $stderr.puts "Error creating collection for #{uuid}: #{error}"
        return false
      end

      begin
        # update src_collection collection name, pdh, and manifest_text
        src_collection.name = (src_collection.name || '') + ' (' + (reason || '') + '; salvaged data at ' + new_collection.uuid + ')'
        src_collection.manifest_text = ''
        src_collection.portable_data_hash = 'd41d8cd98f00b204e9800998ecf8427e+0'
        src_collection.save!
        $stderr.puts "Collection #{uuid} emptied and renamed to #{src_collection.name.inspect}."
      rescue => error
        $stderr.puts "Error salvaging collection #{new_collection.uuid}: #{error}"
        return false
      end
    end
  end
end
