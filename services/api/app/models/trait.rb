class Trait < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :properties, Hash

  api_accessible :superuser, :extend => :common do |t|
    t.add :name
    t.add :properties
  end
end
