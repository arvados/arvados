require 'has_uuid'
require 'kind_and_etag'

class BackwardCompatibilityForUserRepositories < ActiveRecord::Migration
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
      super.rpartition("::").last
    end
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

  class Link < ArvadosModel
  end

  class Repository < ArvadosModel
  end

  def up
    remove_index :repositories, name: "repositories_search_index"
    add_index(:repositories, %w(uuid owner_uuid modified_by_client_uuid
                                modified_by_user_uuid name),
              name: "repositories_search_index")
    remove_column :repositories, :fetch_url
    remove_column :repositories, :push_url

    [Link, Log, Repository].each { |m| m.reset_column_information }
    Repository.where("owner_uuid != ?", system_user_uuid).find_each do |repo|
      link_attrs = {
        tail_uuid: repo.owner_uuid,
        link_class: "permission", name: "can_manage", head_uuid: repo.uuid,
      }
      if Link.where(link_attrs).first.nil?
        manage_link = Link.create!(link_attrs)
        Log.log_create(manage_link)
      end
      start_log = Log.log_for(repo)
      repo.owner_uuid = system_user_uuid
      repo.save!
      Log.log_update(repo, start_log)
    end
  end

  def down
    raise ActiveRecord::IrreversibleMigration.
      new("can't restore prior fetch and push URLs")
  end
end
