class Node < ArvadosBase
  attr_accessor :object
  def friendly_link_name
    self.hostname
  end
end
