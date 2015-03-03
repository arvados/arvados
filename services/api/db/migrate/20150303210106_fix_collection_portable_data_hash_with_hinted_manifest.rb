class FixCollectionPortableDataHashWithHintedManifest < ActiveRecord::Migration
  include CurrentApiClient

  def each_bad_collection
    Collection.find_each do |coll|
      next unless (coll.manifest_text =~ /\+[B-Z]/)
      stripped_manifest = coll.manifest_text.
        gsub(/( [0-9a-f]{32}(\+\d+)?)(\+\S+)/, '\1')
      stripped_pdh = sprintf("%s+%i",
                             Digest::MD5.hexdigest(stripped_manifest),
                             stripped_manifest.bytesize)
      yield [coll, stripped_pdh] if (coll.portable_data_hash != stripped_pdh)
    end
  end

  def up
    copied_attr_names =
      [:owner_uuid, :created_at, :modified_by_client_uuid,
       :modified_by_user_uuid, :modified_at, :updated_at, :name,
       :description, :portable_data_hash, :replication_desired,
       :replication_confirmed, :replication_confirmed_at]
    new_expiry = Date.new(2038, 1, 31)

    act_as_system_user
    each_bad_collection do |coll, stripped_pdh|
      # Create a copy of the collection including bad portable data hash,
      # with an expiration.  This makes it possible to resolve the bad
      # portable data hash, but the expiration can hide the Collection
      # from more user-friendly interfaces like Workbench.
      properties = coll.properties.dup
      properties["migrated_from"] ||= coll.uuid
      coll_copy = Collection.
        create!(manifest_text: coll.manifest_text,
                properties: properties,
                expires_at: coll.expires_at || new_expiry)
      # update_column skips validations and callbacks, which lets us
      # set an "invalid" portable_data_hash and avoid messing with
      # modification metadata.
      copied_attr_names.each do |attr_sym|
        coll_copy.update_column(attr_sym, coll.send(attr_sym))
      end
      coll.update_column(:portable_data_hash, stripped_pdh)
    end
  end

  def down
    act_as_system_user
    each_bad_collection do |coll, stripped_pdh|
      if ((src_uuid = coll.properties["migrated_from"]) and
          (src_coll = Collection.where(uuid: src_uuid).first) and
          (src_coll.portable_data_hash == stripped_pdh))
        src_coll.update_column(:portable_data_hash, coll.portable_data_hash)
      end
      coll.delete
    end
  end
end
