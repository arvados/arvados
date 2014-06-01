class Link < ArvadosBase
  attr_accessor :head
  attr_accessor :tail
  def self.by_tail(t, opts={})
    where(opts.merge :tail_uuid => t.uuid)
  end

  def default_name
    self.class.resource_class_for_uuid(head_uuid).default_name rescue super
  end
end
