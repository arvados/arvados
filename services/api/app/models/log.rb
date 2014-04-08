class Log < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :properties, Hash
  before_validation :set_default_event_at
  before_save { self.owner_uuid = self.system_user_uuid }
  attr_accessor :object

  api_accessible :user, extend: :common do |t|
    t.add :object_kind
    t.add :object_uuid
    t.add :object, :if => :object
    t.add :event_at
    t.add :event_type
    t.add :summary
    t.add :properties
  end

  def self.start_from(thing, event_type)
    self.new do |log|
      log.event_type = event_type
      log.properties = {
        'old_etag' => nil,
        'old_attributes' => nil,
      }
      log.seed_basics_from thing
    end
  end

  def update_to(thing)
    self.seed_basics_from thing
    self.properties["new_etag"] = thing.andand.etag
    self.properties["new_attributes"] = thing.andand.attributes
    case self.event_type
    when "create"
      self.event_at = thing.created_at
    when "update"
      self.event_at = thing.modified_at
    when "destroy"
      self.event_at = Time.now
    end
  end

  def seed_basics_from(thing)
    if not thing.nil?
      self.object_kind ||= thing.kind
      self.object_uuid ||= thing.uuid
      self.summary ||= "#{self.event_type} of #{thing.uuid}"
    end
  end

  protected

  def permission_to_create
    true
  end

  def permission_to_update
    false
  end

  def permission_to_destroy
    false
  end

  def set_default_event_at
    self.event_at ||= Time.now
  end
end
