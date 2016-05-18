class JobTask < ArvadosBase
  def work_unit(label="")
    JobTaskWorkUnit.new(self, label)
  end
end
