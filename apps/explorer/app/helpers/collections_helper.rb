module CollectionsHelper
  def d3ify_links(links)
    links.collect do |x|
      {source: x.tail_uuid, target: x.head_uuid, type: x.name}
    end
  end
end
