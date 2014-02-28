class Collection < ArvadosBase

  def total_bytes
    if files
      tot = 0
      files.each do |file|
        tot += file[2]
      end
      tot
    end
  end

  def attribute_editable?(attr)
    false
  end

  def self.creatable?
    false
  end

  def provenance
    $arvados_api_client.api "collections/#{self.uuid}/", "provenance"
  end

  def used_by
    $arvados_api_client.api "collections/#{self.uuid}/", "used_by"
  end

  # def selection_label
  #   name = ''
  #   i = 0 
  #   if self.files.length > 3
  #     m = 3
  #   else
  #     m = self.files.length 
  #   end
  #   while i < m
  #     name += "#{self.files[i][1]}"
  #     i += 1
  #     name += ", " if i < m
  #   end
  #   if i < self.files.length
  #     name += "&ellip;"
  #   end
  #   name
  # end
end
