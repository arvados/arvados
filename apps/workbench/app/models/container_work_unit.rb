class ContainerWorkUnit < ProxyWorkUnit
  def uri
    uuid = get(:uuid)
    "/containers/#{uuid}"
  end

  def title
    "container"
  end
end
