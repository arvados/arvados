class User < ArvadosBase
  def initialize(*args)
    super(*args)
    @attribute_sortkey['first_name'] = '050'
    @attribute_sortkey['last_name'] = '051'
  end

  def self.current
    res = $arvados_api_client.api self, '/current'
    $arvados_api_client.unpack_api_response(res)
  end
end
