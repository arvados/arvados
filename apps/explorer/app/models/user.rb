class User < OrvosBase
  def self.current
    res = $orvos_api_client.api self, '/current'
    $orvos_api_client.unpack_api_response(res)
  end
end
