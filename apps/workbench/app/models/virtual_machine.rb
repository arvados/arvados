class VirtualMachine < ArvadosBase
  def self.creatable?
    current_user.andand.is_admin
  end
end
