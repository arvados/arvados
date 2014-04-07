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

  protected

  def set_default_event_at
    self.event_at ||= Time.now
  end
end
