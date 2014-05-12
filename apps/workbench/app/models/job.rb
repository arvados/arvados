class Job < ArvadosBase
  def attribute_editable?(attr)
    false
  end

  def self.creatable?
    false
  end
end
