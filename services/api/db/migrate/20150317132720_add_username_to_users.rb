require 'has_uuid'
require 'kind_and_etag'

class AddUsernameToUsers < ActiveRecord::Migration
  include CurrentApiClient

  SEARCH_INDEX_COLUMNS =
    ["uuid", "owner_uuid", "modified_by_client_uuid",
     "modified_by_user_uuid", "email", "first_name", "last_name",
     "identity_url", "default_owner_uuid"]

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

  class User < ArvadosModel
  end

  def sanitize_username(username)
    username.
      sub(/^[^A-Za-z]+/, "").
      gsub(/[^A-Za-z0-9]/, "")
  end

  def usernames_wishlist(user)
    usernames = Hash.new(0)
    usernames[user.email.split("@", 2).first] += 1
    Link.
       where(tail_uuid: user.uuid, link_class: "permission", name: "can_login").
       find_each do |login_perm|
      username = login_perm.properties["username"]
      usernames[username] += 2 if (username and not username.empty?)
    end
    usernames.keys.
      sort_by { |n| -usernames[n] }.
      map { |n| sanitize_username(n) }.
      reject(&:empty?)
  end

  def increment_username(username)
    @username_suffixes[username] += 1
    "%s%i" % [username, @username_suffixes[username]]
  end

  def each_wanted_username(user)
    usernames = usernames_wishlist(user)
    usernames.each { |n| yield n }
    base_username = usernames.first || "arvadosuser"
    loop { yield increment_username(base_username) }
  end

  def recreate_search_index(columns)
    remove_index :users, name: "users_search_index"
    add_index :users, columns, name: "users_search_index"
  end

  def up
    @username_suffixes = Hash.new(1)
    add_column :users, :username, :string, null: true
    add_index :users, :username, unique: true
    recreate_search_index(SEARCH_INDEX_COLUMNS + ["username"])

    [Link, Log, User].each { |m| m.reset_column_information }
    User.validates(:username, uniqueness: true, allow_nil: true)
    User.where(is_active: true).order(created_at: :asc).find_each do |user|
      start_log = Log.log_for(user)
      each_wanted_username(user) do |username|
        user.username = username
        break if user.valid?
      end
      user.save!
      Log.log_update(user, start_log)
    end
  end

  def down
    remove_index :users, :username
    recreate_search_index(SEARCH_INDEX_COLUMNS)
    remove_column :users, :username
  end
end
