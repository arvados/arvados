class AuthorizedKey < ArvadosModel
  include HasUuid
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
      valid_key = SSHKey.valid_ssh_public_key? self.public_key

      if not valid_key
        errors.add(:public_key, "does not appear to be a valid ssh-rsa or dsa public key")
      else
        # Valid if no other rows have this public key
        if self.class.where('uuid != ? and public_key like ?',
                            uuid || '', "%#{self.public_key}%").any?
          errors.add(:public_key, "already exists in the database, use a different key.")
          return false
        end
      end
    end
    return true
  end
end
