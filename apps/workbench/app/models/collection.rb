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
end
