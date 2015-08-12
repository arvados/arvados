#!/usr/bin/env ruby

# Take two input parameters: a collection uuid and reason
# Get "src" collection with the given uuid
# Create a new collection "dest" with:
#   src.manifest_text as "invalid manifest_text.txt"
#   Locators from src.manifest_text as "salvaged_data"
# Update src collection:
#   Set src.manifest_text to: ""
#   Append to src.name: " (reason; salvaged data at dest.uuid)"
#   Set portable_data_hash to "d41d8cd98f00b204e9800998ecf8427e+0"

require 'trollop'

opts = Trollop::options do
  banner ''
  banner "Usage: salvage_collection.rb " +
    "{uuid} {reason}"
  banner ''
  opt :uuid, <<-eos
uuid of the collection to be salvaged.
  eos
  opt :reason, <<-eos
Reason for salvaging.
  eos
end

if ARGV.count < 1
  Trollop::die "required uuid argument is missing"
end

uuid, reason = ARGV

require File.dirname(__FILE__) + '/../config/environment'
require 'arvados/keep'
include ApplicationHelper

def salvage_collection uuid, reason
  act_as_system_user do
    root_dir = '/tmp/salvage_uuids'
    Dir.mkdir(root_dir) unless File.exists?(root_dir)

    src = Collection.find_by_uuid uuid
    if !src
      puts "No collection found for #{uuid}"
      return
    end

    begin
      src_manifest_text = src.manifest_text || ''

      # write the manifest_text to a file
      dir = root_dir+"/"+uuid
      Dir.mkdir(dir) unless File.exists?(dir)
      File.write(dir+"/invalid_manifest_text.txt", src_manifest_text)

      # also, create another file with the locators from the collection manifest_text
      locators = []
      src_manifest_text.each_line do |line|
        line.split(' ').each do |word|
          if match = Keep::Locator::LOCATOR_REGEXP.match(word)
            word = word.split('+')[0..1].join('+')  # get rid of any hints
            locators << word
          end
        end
      end

      locators_str = locators.join(' ')
      File.write(dir+"/salvaged_data", locators_str)

=begin
      # create new collection with salvaged data
      dest_manifest_text = ". "
      dest_manifest_text += (src.portable_data_hash + " 0:#{src_manifest_text.length}:invalid_manifest_text.txt\n")
      dest_manifest_text += (". " + locators_str + " 0:#{locators_str.length}:salvaged_data\n")
      dest_name = "Salvaged from " + uuid + ", " + src.portable_data_hash
      dest = Collection.new name: dest_name, manifest_text: dest_manifest_text
      dest.save!
=end

      # create new collection with salvaged data using 'arv keep put'
      created = %x(arv keep put #{dir}/*)
      created.rstrip!
      match = created.match HasUuid::UUID_REGEX
      raise "uuid not found" if !match
      puts "Created salvaged collection for #{uuid} with uuid: #{created}  #{match}"
    rescue => error
      puts "Error creating salvaged collection for #{uuid}: #{error}"
      return
    end

    begin
      # update src collection name, pdh, and manifest_text
      src.name = (src.name || '') + ' (' + (reason || '') + '; salvaged data at ' + created + ')'
      src.manifest_text = ''
      src.portable_data_hash = 'd41d8cd98f00b204e9800998ecf8427e+0'
      src.save!
      puts "Updated collection #{uuid}"
    rescue => error
      puts "Error updating source collection #{uuid}: #{error}"
    end
  end
end

# Salvage the collection with the given uuid
salvage_collection uuid, reason
