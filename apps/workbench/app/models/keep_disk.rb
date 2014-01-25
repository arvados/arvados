class KeepDisk < ArvadosBase
  def self.creatable?
    current_user and current_user.is_admin
  end
end
