module ContainerRequestsHelper

  # yields collection id (pdh or uuid), and full file_path
  def cr_input_collections(path, &b)
    case path
    when ArvadosBase
      path.class.columns.each do |c|
        cr_input_collections(path[c.name.to_sym], &b)
      end
    when Hash
      path.each do |k, v|
        cr_input_collections(v, &b)
      end
    when Array
      path.each do |v|
        cr_input_collections(v, &b)
      end
    when String
      if m = /[a-f0-9]{32}\+\d+/.match(path)
        yield m[0], path.split('keep:')[-1]
      elsif m = /[0-9a-z]{5}-4zz18-[0-9a-z]{15}/.match(path)
        yield m[0], path.split('keep:')[-1]
      end
    end
  end
end
