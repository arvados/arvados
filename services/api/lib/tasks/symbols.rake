# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'current_api_client'

include CurrentApiClient

def has_symbols? x
  if x.is_a? Hash
    x.each do |k,v|
      return true if has_symbols?(k) or has_symbols?(v)
    end
  elsif x.is_a? Array
    x.each do |k|
      return true if has_symbols?(k)
    end
  elsif x.is_a? Symbol
    return true
  elsif x.is_a? String
    return true if x.start_with?(':') && !x.start_with?('::')
  end
  false
end

def check_for_serialized_symbols rec
  jsonb_cols = rec.class.columns.select{|c| c.type == :jsonb}.collect{|j| j.name}
  (jsonb_cols + rec.class.serialized_attributes.keys).uniq.each do |colname|
    if has_symbols? rec.attributes[colname]
      st = recursive_stringify rec.attributes[colname]
      puts "Found value potentially containing Ruby symbols in #{colname} attribute of #{rec.uuid}, current value is\n#{rec.attributes[colname].to_s[0..1024]}\nrake symbols:stringify will update it to:\n#{st.to_s[0..1024]}\n\n"
    end
  end
end

def recursive_stringify x
  if x.is_a? Hash
    Hash[x.collect do |k,v|
           [recursive_stringify(k), recursive_stringify(v)]
         end]
  elsif x.is_a? Array
    x.collect do |k|
      recursive_stringify k
    end
  elsif x.is_a? Symbol
    x.to_s
  elsif x.is_a? String and x.start_with?(':') and !x.start_with?('::')
    x[1..-1]
  else
    x
  end
end

def stringify_serialized_symbols rec
  # ensure_serialized_attribute_type should prevent symbols from
  # getting into the database in the first place. If someone managed
  # to get them into the database (perhaps using an older version)
  # we'll convert symbols to strings when loading from the
  # database. (Otherwise, loading and saving an object with existing
  # symbols in a serialized field will crash.)
  jsonb_cols = rec.class.columns.select{|c| c.type == :jsonb}.collect{|j| j.name}
  (jsonb_cols + rec.class.serialized_attributes.keys).uniq.each do |colname|
    if has_symbols? rec.attributes[colname]
      begin
        st = recursive_stringify rec.attributes[colname]
        puts "Updating #{colname} attribute of #{rec.uuid} from\n#{rec.attributes[colname].to_s[0..1024]}\nto\n#{st.to_s[0..1024]}\n\n"
        rec.write_attribute(colname, st)
        rec.save!
      rescue => e
        puts "Failed to update #{rec.uuid}: #{e}"
      end
    end
  end
end

namespace :symbols do
  desc 'Warn about serialized values starting with ":" that may be symbols'
  task check: :environment do
    [ApiClientAuthorization, ApiClient,
     AuthorizedKey, Collection,
     Container, ContainerRequest, Group,
     Human, Job, JobTask, KeepDisk, KeepService, Link,
     Node, PipelineInstance, PipelineTemplate,
     Repository, Specimen, Trait, User, VirtualMachine,
     Workflow].each do |klass|
      act_as_system_user do
        klass.all.each do |c|
          check_for_serialized_symbols c
        end
      end
    end
  end

  task stringify: :environment do
    [ApiClientAuthorization, ApiClient,
     AuthorizedKey, Collection,
     Container, ContainerRequest, Group,
     Human, Job, JobTask, KeepDisk, KeepService, Link,
     Node, PipelineInstance, PipelineTemplate,
     Repository, Specimen, Trait, User, VirtualMachine,
     Workflow].each do |klass|
      act_as_system_user do
        klass.all.each do |c|
          stringify_serialized_symbols c
        end
      end
    end
  end
end
