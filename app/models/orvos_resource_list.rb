class OrvosResourceList
  def initialize(resource_class)
    @resource_class = resource_class
  end

  def eager(bool=true)
    @eager = bool
    self
  end

  def where(cond)
    cond = cond.dup
    cond.keys.each do |uuid_key|
      if cond[uuid_key] and (cond[uuid_key].is_a? Array or
                             cond[uuid_key].is_a? OrvosBase)
        # Coerce cond[uuid_key] to an array of uuid strings.  This
        # allows caller the convenience of passing an array of real
        # objects and uuids in cond[uuid_key].
        if !cond[uuid_key].is_a? Array
          cond[uuid_key] = [cond[uuid_key]]
        end
        cond[uuid_key] = cond[uuid_key].collect do |item|
          if item.is_a? OrvosBase
            item.uuid
          else
            item
          end
        end
      end
    end
    cond.keys.select { |x| x.match /_kind$/ }.each do |kind_key|
      if cond[kind_key].is_a? Class
        cond = cond.merge({ kind_key => 'orvos#' + $orvos_api_client.class_kind(cond[kind_key]) })
      end
    end
    res = $orvos_api_client.api @resource_class, '', {
      _method: 'GET',
      where: cond,
      eager: (@eager ? '1' : '0')
    }
    @results = $orvos_api_client.unpack_api_response res
  end

  def all
    res = $orvos_api_client.api @resource_class, '', {
      _method: 'GET',
      eager: (@eager ? '1' : '0')
    }
    @results = $orvos_api_client.unpack_api_response res
  end

  def to_ary
    @results
  end
end
