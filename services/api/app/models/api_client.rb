class ApiClient < ActiveRecord::Base
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  has_many :api_client_authorizations

  api_accessible :superuser, :extend => :common do |t|
    t.add :name
    t.add :url_prefix
    t.add :is_trusted
  end
end
