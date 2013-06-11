class AuthorizedKey < ArvadosBase
  def attribute_editable?(attr)
    if attr.to_s == 'authorized_user_uuid'
      current_user and current_user.is_admin
    else
      super(attr)
    end
  end
end
