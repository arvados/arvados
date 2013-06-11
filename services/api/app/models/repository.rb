class Repository < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate

  api_accessible :superuser, :extend => :common do |t|
    t.add :name
    t.add :fetch_url
    t.add :push_url
  end

  protected

  def permission_to_create
    current_user and current_user.is_admin
  end
  def permission_to_update
    current_user and current_user.is_admin
  end
end
