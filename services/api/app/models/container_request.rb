class ContainerRequest < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  serialize :properties, Hash
  serialize :environment, Hash
  serialize :mounts, Hash
  serialize :runtime_constraints, Hash
  serialize :command, Array

  before_create :set_state_before_save

  api_accessible :user, extend: :common do |t|
    t.add :command
    t.add :container_count_max
    t.add :container_image
    t.add :container_uuid
    t.add :cwd
    t.add :description
    t.add :environment
    t.add :expires_at
    t.add :filters
    t.add :mounts
    t.add :name
    t.add :output_path
    t.add :priority
    t.add :properties
    t.add :requesting_container_uuid
    t.add :runtime_constraints
    t.add :state
  end

  # Supported states for a container request
  States =
    [
     (Uncommitted = 'Uncommitted'),
     (Committed = 'Committed'),
     (Final = 'Final'),
    ]

  def set_state_before_save
    self.state ||= Uncommitted
  end



end
