class Human < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :properties, Hash

  api_accessible :user, extend: :common do |t|
    t.add :properties
  end
end
