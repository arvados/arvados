class ApiClientAuthorization < ArvadosBase
  def attribute_editable?(attr)
    ['expires_at', 'default_owner'].index attr
  end
  def self.creatable?
    false
  end
end
