class JobTask < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :parameters, Hash

  api_accessible :superuser, :extend => :common do |t|
    t.add :job_uuid
    t.add :sequence
    t.add :parameters
    t.add :output
    t.add :progress
    t.add :success
  end
end
