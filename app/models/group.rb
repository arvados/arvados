class Group < OrvosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate

  api_accessible :superuser, :extend => :common do |t|
    t.add :name
    t.add :description
  end
end
