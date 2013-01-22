class Log < ActiveRecord::Base
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :info, Hash
  before_validation :set_default_event_at

  api_accessible :superuser, :extend => :common do |t|
    t.add :object_kind
    t.add :object_uuid
    t.add :event_at
    t.add :event_type
    t.add :summary
    t.add :info
  end

  protected

  def set_default_event_at
    self.event_at ||= Time.now
  end
end
