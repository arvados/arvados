class Metadatum < ActiveRecord::Base
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :info, Hash
  before_validation :populate_native_target

  def self.add_schema_columns
    [ { name: :head_kind, type: :string },
      { name: :head_uuid, type: :string } ]
  end

  api_accessible :superuser, :extend => :common do |t|
    t.add :target_kind
    t.add :target_uuid
    t.add :metadata_class
    t.add :key
    t.add :value
    t.add :info
    t.add :head_kind
    t.add :head_uuid
  end

  def info
    @info ||= Hash.new
    super
  end

  def head_kind
    @head_kind if populate_head_object
  end

  def head_uuid
    @head_uuid if populate_head_object
  end

  protected

  def populate_head_object
    @head_object ||= begin
      @head_kind = self.value.
        sub(/^(.*)#.*/,'\1')
      logger.debug @head_kind
      class_name = @head_kind.
        sub(/^orvos#/,'').
        pluralize.
        classify
      logger.debug "class_name is #{class_name}"
      @head_uuid = self.value.split('#').last
      logger.debug "uuid is @head_uuid"
      @head_object = class_name.
        constantize.
        where('uuid = ?', @head_uuid).
        first
      @head_object
    rescue
      @head_kind = nil
      @head_uuid = nil
      false
    end || false unless @head_object == false
  end

  def populate_native_target
    begin
      class_name = target_kind.
        sub(/^orvos#/,'').
        pluralize.
        classify
      self.native_target_type = class_name
      self.native_target_id = class_name.
        constantize.
        where('uuid = ?', target_uuid).
        first.
        id
    rescue
      self.native_target_type = nil
      self.native_target_id = nil
    end
  end
end
