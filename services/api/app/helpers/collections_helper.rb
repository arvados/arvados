module CollectionsHelper
  def stripped_portable_data_hash(uuid)
    m = /([a-f0-9]{32}(\+[0-9]+)?)(\+.*)?/.match(uuid)
    if m
      m[1]
    else
      nil
    end
  end
end
