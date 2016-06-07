class JobTask < ArvadosBase
  def work_unit(label=nil)
    JobTaskWorkUnit.new(self, label)
  end
end
