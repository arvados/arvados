class Metadatum < OrvosBase
  def self.by_target(t, opts={})
    where(opts.merge :target_kind => t.kind, :target_uuid => t.uuid)
  end
end
