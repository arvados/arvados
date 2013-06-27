class Human < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :properties, Hash

  api_accessible :superuser, :extend => :common do |t|
    t.add :properties
  end
end
