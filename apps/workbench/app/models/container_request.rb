class ContainerRequest < ArvadosBase
  def cancel
    arvados_api_client.api "container_requests/#{self.uuid}/", "cancel", {}
  end

  def work_unit(label=nil)
    ContainerWorkUnit.new(self, label)
  end
end
