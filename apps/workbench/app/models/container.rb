class Container < ArvadosBase
  def self.creatable?
    false
  end

  def work_unit(label=nil, child_objects=nil)
    ContainerWorkUnit.new(self, label, self.uuid, child_objects=child_objects)
  end
end
