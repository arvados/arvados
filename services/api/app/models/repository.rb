class Repository < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :fetch_url
    t.add :push_url
  end

  def push_url
    super || self.name && "git@git.#{Rails.configuration.uuid_prefix}.arvadosapi.com:#{self.name}.git"
  end

  def fetch_url
    super || push_url
  end

  protected

  def permission_to_create
    current_user and current_user.is_admin
  end
  def permission_to_update
    current_user and current_user.is_admin
  end
end
