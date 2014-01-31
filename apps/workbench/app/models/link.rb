class Link < ArvadosBase
  attr_accessor :head
  attr_accessor :tail
  def self.by_tail(t, opts={})
    where(opts.merge :tail_kind => t.kind, :tail_uuid => t.uuid)
  end

  def friendly_link_name
    "(#{link_class}) #{tail_kind.sub 'arvados#', ' '} #{name} #{head_kind.sub 'arvados#', ' '}"
  end
end
