class User < ActiveRecord::Base
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :prefs, Hash
  has_many :api_client_authorizations

  api_accessible :superuser, :extend => :common do |t|
    t.add :email
    t.add :full_name
    t.add :first_name
    t.add :last_name
    t.add :identity_url
    t.add :is_admin
    t.add :prefs
  end

  def full_name
    "#{first_name} #{last_name}"
  end

end
