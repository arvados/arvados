class Metadatum < ActiveRecord::Base
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :info, Hash

  api_accessible :superuser, :extend => :common do |t|
    t.add :tail_kind
    t.add :tail
    t.add :metadata_class
    t.add :name
    t.add :head_kind
    t.add :head
    t.add :info
  end

  def info
    @info ||= Hash.new
    super
  end
end
