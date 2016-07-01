class Container < ArvadosBase
  def self.creatable?
    false
  end

  def work_unit(label=nil)
    ContainerWorkUnit.new(self, label)
  end
end
