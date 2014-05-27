class Job < ArvadosBase
  def self.goes_in_folders?
    true
  end

  def attribute_editable?(attr)
    false
  end

  def self.creatable?
    false
  end
end
