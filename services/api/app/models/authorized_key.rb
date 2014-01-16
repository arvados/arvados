class AuthorizedKey < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate
  before_create :permission_to_set_authorized_user_uuid
  before_update :permission_to_set_authorized_user_uuid

  belongs_to :authorized_user, :foreign_key => :authorized_user_uuid, :class_name => 'User', :primary_key => :uuid

  validate :public_key_must_be_unique

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

  def public_key_must_be_unique
    if self.public_key
      key = /ssh-rsa [A-Za-z0-9+\/]+/.match(self.public_key)
      
      if not key
        errors.add(:public_key, "Does not appear to be a valid ssh-rsa key")
      else
        # Valid if no other rows have this public key
        if self.class.where('public_key like ?', "%#{key[0]}%").any?
          errors.add(:public_key, "Key already exists in the database, use a different key.")
          return false
        end
      end
    end
    return true
  end
end
