class Group < ArvadosBase
  def owned_items params={}
    res = $arvados_api_client.api self.class, "/#{self.uuid}/owned_items", {
      _method: 'GET'
    }.merge(params)
    ret = ArvadosResourceList.new
    ret.results = $arvados_api_client.unpack_api_response(res)
    ret
  end
end
