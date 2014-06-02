class AuthorizedKey < ArvadosBase
  def attribute_editable? attr, *args
    if attr.to_s == 'authorized_user_uuid'
      current_user and current_user.is_admin
    else
      super attr, *args
    end
  end
end
