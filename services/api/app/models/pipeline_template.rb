class PipelineTemplate < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :components, Hash

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :components
  end
end
