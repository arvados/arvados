class User < ArvadosBase
  def self.current
    res = $arvados_api_client.api self, '/current'
    $arvados_api_client.unpack_api_response(res)
  end
end
