module CollectionsHelper
  def d3ify_links(links)
    links.collect do |x|
      {source: x.tail_uuid, target: x.head_uuid, type: x.name}
    end
  end

  def self.match(uuid)
    /^([a-f0-9]{32}(\+[0-9]+)?)(\+.*)?$/.match(uuid.to_s)
  end
end
