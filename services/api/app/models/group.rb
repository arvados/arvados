class Group < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :description
  end
end
