# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'has_uuid'
require 'kind_and_etag'

class FixCollectionPortableDataHashWithHintedManifest < ActiveRecord::Migration[4.2]
  include CurrentApiClient

  class ArvadosModel < ActiveRecord::Base
    self.abstract_class = true
    extend HasUuid::ClassMethods
    include CurrentApiClient
    include KindAndEtag
    before_create do |record|
      record.uuid ||= record.class.generate_uuid
      record.owner_uuid ||= system_user_uuid
    end
    serialize :properties, Hash

    def self.to_s
      # Clean up the name of the stub model class so we generate correct UUIDs.
      super.sub("FixCollectionPortableDataHashWithHintedManifest::", "")
    end
  end

  class Collection < ArvadosModel
  end

  class Log < ArvadosModel
    def self.log_for(thing, age="old")
      { "#{age}_etag" => thing.etag,
        "#{age}_attributes" => thing.attributes,
      }
    end

    def self.log_create(thing)
      new_log("create", thing, log_for(thing, "new"))
    end

    def self.log_update(thing, start_state)
      new_log("update", thing, start_state.merge(log_for(thing, "new")))
    end

    def self.log_destroy(thing)
      new_log("destroy", thing, log_for(thing, "old"))
    end

    private

    def self.new_log(event_type, thing, properties)
      create!(event_type: event_type,
              event_at: Time.now,
              object_uuid: thing.uuid,
              object_owner_uuid: thing.owner_uuid,
              properties: properties)
    end
  end

  def each_bad_collection
    end_coll = Collection.order("id DESC").first
    return if end_coll.nil?
    seen_uuids = []
    ("A".."Z").each do |hint_char|
      query = Collection.
        where("id <= ? AND manifest_text LIKE '%+#{hint_char}%'", end_coll.id)
      unless seen_uuids.empty?
        query = query.where("uuid NOT IN (?)", seen_uuids)
      end
      # It's important to make sure that this line doesn't swap.  The
      # worst case scenario is that it finds a batch of collections that
      # all have maximum size manifests (64MiB).  With a batch size of
      # 50, that's about 3GiB.  Figure it will end up being 4GiB after
      # other ActiveRecord overhead.  That's a size we're comfortable with.
      query.find_each(batch_size: 50) do |coll|
        seen_uuids << coll.uuid
        stripped_manifest = coll.manifest_text.
          gsub(/( [0-9a-f]{32}(\+\d+)?)\+\S+/, '\1')
        stripped_pdh = sprintf("%s+%i",
                               Digest::MD5.hexdigest(stripped_manifest),
                               stripped_manifest.bytesize)
        yield [coll, stripped_pdh] if (coll.portable_data_hash != stripped_pdh)
      end
    end
  end

  def up
    Collection.reset_column_information
    Log.reset_column_information
    copied_attr_names =
      [:owner_uuid, :created_at, :modified_by_client_uuid, :manifest_text,
       :modified_by_user_uuid, :modified_at, :updated_at, :name,
       :description, :portable_data_hash, :replication_desired,
       :replication_confirmed, :replication_confirmed_at, :expires_at]
    new_expiry = Date.new(2038, 1, 31)

    each_bad_collection do |coll, stripped_pdh|
      # Create a copy of the collection including bad portable data hash,
      # with an expiration.  This makes it possible to resolve the bad
      # portable data hash, but the expiration can hide the Collection
      # from more user-friendly interfaces like Workbench.
      start_log = Log.log_for(coll)
      attributes = Hash[copied_attr_names.map { |key| [key, coll.send(key)] }]
      attributes[:expires_at] ||= new_expiry
      attributes[:properties] = (coll.properties.dup rescue {})
      attributes[:properties]["migrated_from"] ||= coll.uuid
      coll_copy = Collection.create!(attributes)
      Log.log_create(coll_copy)
      coll.update(portable_data_hash: stripped_pdh)
      Log.log_update(coll, start_log)
    end
  end

  def down
    Collection.reset_column_information
    Log.reset_column_information
    each_bad_collection do |coll, stripped_pdh|
      if ((src_uuid = coll.properties["migrated_from"]) and
          (src_coll = Collection.where(uuid: src_uuid).first) and
          (src_coll.portable_data_hash == stripped_pdh))
        start_log = Log.log_for(src_coll)
        src_coll.portable_data_hash = coll.portable_data_hash
        src_coll.save!
        Log.log_update(src_coll, start_log)
        coll.destroy or raise Exception.new("failed to destroy old collection")
        Log.log_destroy(coll)
      end
    end
  end
end
