class ContainerRequest < ArvadosBase
  def self.creatable?
    false
  end

  def textile_attributes
    [ 'description' ]
  end

  def self.goes_in_projects?
    true
  end

  def work_unit(label=nil)
    ContainerWorkUnit.new(self, label, self.uuid)
  end
end
