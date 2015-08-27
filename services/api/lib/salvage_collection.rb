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
  include ApplicationHelper
  require 'tempfile'
  require 'shellwords'

  def self.salvage_collection_arv_put cmd
    new_manifest = %x(#{cmd})
    if $?.success?
      new_manifest
    else
      raise "Error during arv-put: #{$?} (cmd was #{cmd.inspect})"
    end
  end

  # Get all the locators from the original manifest
  LOCATOR_REGEXP = /((.*))?([[:xdigit:]]{32})(\+(.*))?\z/
  def self.salvage_collection_locator_data manifest
      locators = []
      size = 0
      manifest.each_line do |line|
        line.split(' ').each do |word|
          if match = LOCATOR_REGEXP.match(word)
            if match.size == 6 and match[5]
              size_str = match[5].split('+')[0]
              if size_str.to_i.to_s == size_str
                word = match[3] + '+' + size_str     # get rid of any other hints
                size += size_str.to_i
              else
                word = match[3]
              end
            else
              word = match[3]
            end
            locators << word
          end
        end
      end
      locators << 'd41d8cd98f00b204e9800998ecf8427e+0' if !locators.any?
      return [locators, size]
  end

  def self.salvage_collection uuid, reason='salvaged - see #6277, #6859'
    act_as_system_user do
      if !ENV['ARVADOS_API_TOKEN'].present? or !ENV['ARVADOS_API_HOST'].present?
        raise "ARVADOS environment variables missing. Please set your admin user credentials as ARVADOS environment variables."
      end

      if !uuid.present?
        raise "Collection UUID is required."
      end

      src_collection = Collection.find_by_uuid uuid
      if !src_collection
        raise "No collection found for #{uuid}."
      end

      src_manifest = src_collection.manifest_text || ''

      # create new collection using 'arv-put' with original manifest_text as the data
      temp_file = Tempfile.new('temp')
      temp_file.write(src_manifest)

      temp_file.close

      new_manifest = salvage_collection_arv_put "arv-put --as-stream --use-filename invalid_manifest_text.txt #{Shellwords::shellescape(temp_file.path)}"

      temp_file.unlink

      # Get the locator data in the format [[locators], size] from the original manifest
      locator_data = salvage_collection_locator_data src_manifest

      new_manifest += (". #{locator_data[0].join(' ')} 0:#{locator_data[1]}:salvaged_data\n")

      new_collection = Collection.new
      new_collection.name = "salvaged from #{src_collection.uuid}, #{src_collection.portable_data_hash}"
      new_collection.manifest_text = new_manifest

      created = new_collection.save!
      raise "New collection creation failed." if !created

      $stderr.puts "Salvaged manifest and data for #{uuid} are in #{new_collection.uuid}."
      puts "Created new collection #{new_collection.uuid}"

      # update src_collection collection name, pdh, and manifest_text
      src_collection.name = (src_collection.name || '') + ' (' + (reason || '') + '; salvaged data at ' + new_collection.uuid + ')'
      src_collection.manifest_text = ''
      src_collection.portable_data_hash = 'd41d8cd98f00b204e9800998ecf8427e+0'
      src_collection.save!
      $stderr.puts "Collection #{uuid} emptied and renamed to #{src_collection.name.inspect}."
    end
  end
end
