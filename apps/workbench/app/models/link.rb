class Link < ArvadosBase
  attr_accessor :head
  attr_accessor :tail
  def self.by_tail(t, opts={})
    where(opts.merge :tail_uuid => t.uuid)
  end

  def default_name
    self.class.resource_class_for_uuid(head_uuid).default_name rescue super
  end

  def self.permissions_for(thing)
    if thing.respond_to? :uuid
      uuid = thing.uuid
    else
      uuid = thing
    end
    result = arvados_api_client.api("permissions", "/#{uuid}")
    arvados_api_client.unpack_api_response(result)
  end
end
