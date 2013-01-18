class Metadatum < OrvosBase
  def self.by_tail(t, opts={})
    where(opts.merge :tail_kind => t.kind, :tail => t.uuid)
  end
end
