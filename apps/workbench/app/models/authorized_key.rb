class AuthorizedKey < ArvadosBase
  def attribute_editable?(attr, ever=nil)
    if (attr.to_s == 'authorized_user_uuid') and (not ever)
      current_user.andand.is_admin
    else
      super
    end
  end

  def self.creatable?
    false
  end
end
