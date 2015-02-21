class ApiClientAuthorization < ArvadosModel
  include KindAndEtag
  include CommonApiTemplate

  belongs_to :api_client
  belongs_to :user
  after_initialize :assign_random_api_token
  serialize :scopes, Array

  api_accessible :user, extend: :common do |t|
    t.add :owner_uuid
    t.add :user_id
    t.add :api_client_id
    t.add :api_token
    t.add :created_by_ip_address
    t.add :default_owner_uuid
    t.add :expires_at
    t.add :last_used_at
    t.add :last_used_by_ip_address
    t.add :scopes
  end

  UNLOGGED_CHANGES = ['last_used_at', 'last_used_by_ip_address', 'updated_at']

  def assign_random_api_token
    self.api_token ||= rand(2**256).to_s(36)
  end

  def owner_uuid
    self.user.andand.uuid
  end
  def owner_uuid_was
    self.user_id_changed? ? User.where(id: self.user_id_was).first.andand.uuid : self.user.andand.uuid
  end
  def owner_uuid_changed?
    self.user_id_changed?
  end

  def uuid
    self.api_token
  end
  def uuid=(x) end
  def uuid_was
    self.api_token_was
  end
  def uuid_changed?
    self.api_token_changed?
  end

  def modified_by_client_uuid
    nil
  end
  def modified_by_client_uuid=(x) end

  def modified_by_user_uuid
    nil
  end
  def modified_by_user_uuid=(x) end

  def modified_at
    nil
  end
  def modified_at=(x) end

  def scopes_allow?(req_s)
    scopes.each do |scope|
      return true if (scope == 'all') or (scope == req_s) or
        ((scope.end_with? '/') and (req_s.start_with? scope))
    end
    false
  end

  def scopes_allow_request?(request)
    scopes_allow? [request.request_method, request.path].join(' ')
  end

  def logged_attributes
    attrs = attributes.dup
    attrs.delete('api_token')
    attrs
  end

  def self.default_orders
    ["#{table_name}.id desc"]
  end

  protected

  def permission_to_create
    current_user.andand.is_admin or (current_user.andand.id == self.user_id)
  end

  def permission_to_update
    (permission_to_create and
     not self.user_id_changed? and
     not self.owner_uuid_changed?)
  end

  def log_update
    super unless (changed - UNLOGGED_CHANGES).empty?
  end
end
