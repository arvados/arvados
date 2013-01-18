class Pipeline < OrvosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :components, Hash

  api_accessible :superuser, :extend => :common do |t|
    t.add :name
    t.add :components
  end
end
