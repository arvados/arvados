class Specimen < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :properties, Hash

  api_accessible :user, extend: :common do |t|
    t.add :material
    t.add :properties
  end

  def properties
    @properties ||= Hash.new
    super
  end
end
