class ApiClientAuthorization < ArvadosBase
  def editable_attributes
    %w(expires_at default_owner_uuid)
  end
  def self.creatable?
    false
  end
end
