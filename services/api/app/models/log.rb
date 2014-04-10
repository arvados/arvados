class Log < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :properties, Hash
  before_validation :set_default_event_at
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

  def fill_object(thing)
    self.object_kind ||= thing.kind
    self.object_uuid ||= thing.uuid
    self.summary ||= "#{self.event_type} of #{thing.uuid}"
    self
  end

  def fill_properties(age, etag_prop, attrs_prop)
    self.properties.merge!({"#{age}_etag" => etag_prop,
                             "#{age}_attributes" => attrs_prop})
  end

  def update_to(thing)
    fill_properties('new', thing.andand.etag, thing.andand.attributes)
    case event_type
    when "create"
      self.event_at = thing.created_at
    when "update"
      self.event_at = thing.modified_at
    when "destroy"
      self.event_at = Time.now
    end
    self
  end

  protected

  def permission_to_create
    true
  end

  def permission_to_update
    current_user.andand.is_admin
  end

  alias_method :permission_to_delete, :permission_to_update

  def set_default_event_at
    self.event_at ||= Time.now
  end

  def log_change(event_type)
    # Don't log changes to logs.
  end
end
