class Group < ArvadosBase
  def self.owned_items
    res = $arvados_api_client.api self, "/#{self.uuid}/owned_items", {}
    $arvados_api_client.unpack_api_response(res)
  end
end
