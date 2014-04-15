module CollectionsHelper
  def d3ify_links(links)
    links.collect do |x|
      {source: x.tail_uuid, target: x.head_uuid, type: x.name}
    end
  end

  def self.match(uuid)
    /^([a-f0-9]{32})(\+[0-9]+)?(\+.*?)?(\/.*)?$/.match(uuid.to_s)
  end

  def self.is_image file
    /\.(jpg|jpeg|gif|png|svg)$/i.match(file)
  end

  def self.file_path file
    f0 = file[0]
    f0 = '' if f0 == '.'
    f0 = f0[2..-1] if f0[0..1] == './'
    f0 += '/' if not f0.empty?
    file_path = "#{f0}#{file[1]}"
  end
end
