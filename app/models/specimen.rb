class Specimen < OrvosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate

  api_accessible :superuser, :extend => :common do |t|
    t.add :material
  end
end
