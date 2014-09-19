class PipelineTemplate < ArvadosBase
  def self.goes_in_projects?
    true
  end

  def self.creatable?
    false
  end

  def textile_attributes
    [ 'description' ]
  end
end
