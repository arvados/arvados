class PipelineTemplate < ArvadosBase
  def self.goes_in_folders?
    true
  end

  def self.creatable?
    false
  end
end
