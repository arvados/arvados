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

  def self.system
    $arvados_system_user ||= begin
                               res = $arvados_api_client.api self, '/system'
                               $arvados_api_client.unpack_api_response(res)
                             end
  end

  def full_name
    (self.first_name || "") + " " + (self.last_name || "")
  end
end
