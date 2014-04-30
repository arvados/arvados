class Group < ArvadosBase
  def contents params={}
    res = $arvados_api_client.api self.class, "/#{self.uuid}/contents", {
      _method: 'GET'
    }.merge(params)
    ret = ArvadosResourceList.new
    ret.results = $arvados_api_client.unpack_api_response(res)
    ret
  end

  def class_for_display
    group_class == 'folder' ? 'Folder' : super
  end
end
