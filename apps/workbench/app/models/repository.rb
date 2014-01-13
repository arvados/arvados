class Repository < ArvadosBase
  def self.creatable?
    current_user and current_user.is_admin
  end
  def attributes_for_display
    super.reject { |x| x[0] == 'fetch_url' }
  end
end
