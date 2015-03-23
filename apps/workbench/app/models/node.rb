class Node < ArvadosBase
  def self.creatable?
    current_user and current_user.is_admin
  end
  def friendly_link_name lookup=nil
    (hostname && !hostname.empty?) ? hostname : uuid
  end
end
