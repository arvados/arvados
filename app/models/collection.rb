class Collection < ActiveRecord::Base
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate

  api_accessible :superuser, :extend => :common do |t|
    t.add :locator
    t.add :portable_data_hash
    t.add :name
    t.add :redundancy
    t.add :redundancy_confirmed_by_client
    t.add :redundancy_confirmed_at
    t.add :redundancy_confirmed_as
  end
end
