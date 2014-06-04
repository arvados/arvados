class Job < ArvadosBase
  def self.goes_in_folders?
    true
  end

  def attribute_editable? attr, *args
    false
  end

  def self.creatable?
    false
  end

  def cancel
    arvados_api_client.api "jobs/#{self.uuid}/", "cancel", {}
  end
end
