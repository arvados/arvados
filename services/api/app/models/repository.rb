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
    return false if not current_user
    return true if current_user.is_admin
    # For normal objects, this is a way to check whether you have
    # write permission. Repositories should be brought closer to the
    # normal permission model during #4253. Meanwhile, we'll
    # special-case this so arv-git-httpd can detect write permission:
    return super if changed_attributes.keys - ['modified_at', 'updated_at'] == []
    false
  end
end
