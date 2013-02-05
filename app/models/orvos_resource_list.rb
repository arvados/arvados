class OrvosResourceList
  def initialize(resource_class)
    @resource_class = resource_class
  end

  def eager(bool=true)
    @eager = bool
    self
  end

  def limit(max_results)
    @limit = max_results
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
    api_params = {
      _method: 'GET',
      where: cond
    }
    api_params[:eager] = '1' if @eager
    api_params[:limit] = @limit if @limit
    res = $orvos_api_client.api @resource_class, '', api_params
    @results = $orvos_api_client.unpack_api_response res
  end

  def all
    where({})
  end

  def to_ary
    @results
  end
end
