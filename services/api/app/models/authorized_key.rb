class AuthorizedKey < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  before_create :permission_to_set_authorized_user_uuid
  before_update :permission_to_set_authorized_user_uuid

  belongs_to :authorized_user, :foreign_key => :authorized_user_uuid, :class_name => 'User', :primary_key => :uuid

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :key_type
    t.add :authorized_user_uuid
    t.add :public_key
    t.add :expires_at
  end

  def permission_to_set_authorized_user_uuid
    # Anonymous users cannot do anything here
    return false if !current_user

    # Administrators can attach a key to any user account
    return true if current_user.is_admin

    # All users can attach keys to their own accounts
    return true if current_user.uuid == authorized_user_uuid

    # Default = deny.
    false
  end
end
