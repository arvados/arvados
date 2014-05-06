class ArvadosResourceList
  include Enumerable

  def initialize resource_class=nil
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

  def offset(skip)
    @offset = skip
    self
  end

  def order(orderby_spec)
    @orderby_spec = orderby_spec
    self
  end

  def filter _filters
    @filters ||= []
    @filters += _filters
    self
  end

  def where(cond)
    cond = cond.dup
    cond.keys.each do |uuid_key|
      if cond[uuid_key] and (cond[uuid_key].is_a? Array or
                             cond[uuid_key].is_a? ArvadosBase)
        # Coerce cond[uuid_key] to an array of uuid strings.  This
        # allows caller the convenience of passing an array of real
        # objects and uuids in cond[uuid_key].
        if !cond[uuid_key].is_a? Array
          cond[uuid_key] = [cond[uuid_key]]
        end
        cond[uuid_key] = cond[uuid_key].collect do |item|
          if item.is_a? ArvadosBase
            item.uuid
          else
            item
          end
        end
      end
    end
    cond.keys.select { |x| x.match /_kind$/ }.each do |kind_key|
      if cond[kind_key].is_a? Class
        cond = cond.merge({ kind_key => 'arvados#' + $arvados_api_client.class_kind(cond[kind_key]) })
      end
    end
    api_params = {
      _method: 'GET',
      where: cond
    }
    api_params[:eager] = '1' if @eager
    api_params[:limit] = @limit if @limit
    api_params[:offset] = @offset if @offset
    api_params[:order] = @orderby_spec if @orderby_spec
    api_params[:filters] = @filters if @filters
    res = $arvados_api_client.api @resource_class, '', api_params
    @results = $arvados_api_client.unpack_api_response res
    self
  end

  def results
    self.where({}) if !@results
    @results
  end

  def results=(r)
    @results = r
  end

  def all
    where({})
  end

  def each(&block)
    results.each do |m|
      block.call m
    end
    self
  end

  def collect
    results.collect do |m|
      yield m
    end
  end

  def first
    results.first
  end

  def last
    results.last
  end

  def [](*x)
    results.send('[]', *x)
  end

  def |(x)
    if x.is_a? Hash
      self.to_hash | x
    else
      results | x.to_ary
    end
  end

  def to_ary
    results
  end

  def to_hash
    Hash[results.collect { |x| [x.uuid, x] }]
  end

  def empty?
    results.empty?
  end

  def items_available
    results.items_available if results.respond_to? :items_available
  end

  def result_limit
    results.limit if results.respond_to? :limit
  end

  def result_offset
    results.offset if results.respond_to? :offset
  end

  def result_links
    results.links if results.respond_to? :links
  end

  # Return links provided with API response that point to the
  # specified object, and have the specified link_class. If link_class
  # is false or omitted, return all links pointing to the specified
  # object.
  def links_for item_or_uuid, link_class=false
    return [] if !result_links
    unless @links_for_uuid
      @links_for_uuid = {}
      result_links.each do |link|
        if link.respond_to? :head_uuid
          @links_for_uuid[link.head_uuid] ||= []
          @links_for_uuid[link.head_uuid] << link
        end
      end
    end
    if item_or_uuid.respond_to? :uuid
      uuid = item_or_uuid.uuid
    else
      uuid = item_or_uuid
    end
    (@links_for_uuid[uuid] || []).select do |link|
      link_class == false or link.link_class == link_class
    end
  end

  # Note: this arbitrarily chooses one of (possibly) multiple names.
  def name_for item_or_uuid
    links_for(item_or_uuid, 'name').first.andand.name
  end

end
