class ContainerRequest < ArvadosBase
  def self.creatable?
    false
  end

  def textile_attributes
    [ 'description' ]
  end

  def work_unit(label=nil)
    ContainerWorkUnit.new(self, label)
  end
end
