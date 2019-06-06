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
  # ensure_serialized_attribute_type should prevent symbols from
  # getting into the database in the first place. If someone managed
  # to get them into the database (perhaps using an older version)
  # we'll convert symbols to strings when loading from the
  # database. (Otherwise, loading and saving an object with existing
  # symbols in a serialized field will crash.)
  jsonb_cols = rec.class.columns.select{|c| c.type == :jsonb}.collect{|j| j.name}
  (jsonb_cols + rec.class.serialized_attributes.keys).uniq.each do |colname|
    if has_symbols? rec.attributes[colname]
      puts "Found string with leading ':' in attribute '#{colname}' of #{rec.uuid}:\n#{rec.attributes[colname].to_s[0..1024]}\n\n"
    end
  end
end

namespace :legacy do
  desc 'Warn about serialized values starting with ":" that may be symbols'
  task symbols: :environment do
    [ApiClientAuthorization, ApiClient,
     AuthorizedKey, Collection,
     Container, ContainerRequest, Group,
     Human, Job, JobTask, KeepDisk, KeepService, Link,
     Node, PipelineInstance, PipelineTemplate,
     Repository, Specimen, Trait, User, VirtualMachine,
     Workflow].each do |klass|
      klass.all.each do |c|
        check_for_serialized_symbols c
      end
    end
  end
end
