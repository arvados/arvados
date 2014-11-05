class ArvadosResourceList
  include ArvadosApiClientHelper
  include Enumerable

  def initialize resource_class=nil
    @resource_class = resource_class
    @fetch_multiple_pages = true
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

  def select(columns=nil)
    # If no column arguments were given, invoke Enumerable#select.
    if columns.nil?
      super()
    else
      @select ||= []
      @select += columns
      self
    end
  end

  def filter _filters
    @filters ||= []
    @filters += _filters
    self
  end

  def where(cond)
    @cond = cond.dup
    @cond.keys.each do |uuid_key|
      if @cond[uuid_key] and (@cond[uuid_key].is_a? Array or
                             @cond[uuid_key].is_a? ArvadosBase)
        # Coerce cond[uuid_key] to an array of uuid strings.  This
        # allows caller the convenience of passing an array of real
        # objects and uuids in cond[uuid_key].
        if !@cond[uuid_key].is_a? Array
          @cond[uuid_key] = [@cond[uuid_key]]
        end
        @cond[uuid_key] = @cond[uuid_key].collect do |item|
          if item.is_a? ArvadosBase
            item.uuid
          else
            item
          end
        end
      end
    end
    @cond.keys.select { |x| x.match /_kind$/ }.each do |kind_key|
      if @cond[kind_key].is_a? Class
        @cond = @cond.merge({ kind_key => 'arvados#' + arvados_api_client.class_kind(@cond[kind_key]) })
      end
    end
    self
  end

  def fetch_multiple_pages(f)
    @fetch_multiple_pages = f
    self
  end

  def results
    if !@results
      @results = []
      self.each_page do |r|
        @results.concat r
      end
    end
    @results
  end

  def results=(r)
    @results = r
    @items_available = r.items_available if r.respond_to? :items_available
    @result_limit = r.limit if r.respond_to? :limit
    @result_offset = r.offset if r.respond_to? :offset
    @result_links = r.links if r.respond_to? :links
    @results
  end

  def all
    results
    self
  end

  def to_ary
    results
  end

  def each_page
    api_params = {
      _method: 'GET'
    }
    api_params[:where] = @cond if @cond
    api_params[:eager] = '1' if @eager
    api_params[:limit] = @limit if @limit
    api_params[:select] = @select if @select
    api_params[:order] = @orderby_spec if @orderby_spec
    api_params[:filters] = @filters if @filters

    item_count = 0

    if @offset
      offset = @offset
    else
      offset = 0
    end

    if @limit.is_a? Integer
      items_to_get = @limit
    else
      items_to_get = 1000000000
    end

    begin
      api_params[:offset] = offset

      res = arvados_api_client.api @resource_class, '', api_params
      items = arvados_api_client.unpack_api_response res

      if items.nil? or items.size == 0
        break
      end

      @items_available = items.items_available if items.respond_to? :items_available
      @result_limit = items.limit
      @result_offset = items.offset
      @result_links = items.links if items.respond_to? :links

      item_count += items.size

      if items.respond_to? :items_available and
          (@limit.nil? or (@limit.is_a? Integer and  @limit > items.items_available))
        items_to_get = items.items_available
      end

      offset = items.offset + items.size

      yield items

    end while @fetch_multiple_pages and item_count < items_to_get
    self
  end

  def each(&block)
    if not @results.nil?
      @results.each &block
    else
      self.each_page do |items|
        items.each do |i|
          block.call i
        end
      end
    end
    self
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

  def to_hash
    Hash[self.collect { |x| [x.uuid, x] }]
  end

  def empty?
    self.first.nil?
  end

  def items_available
    @items_available
  end

  def result_limit
    @result_limit
  end

  def result_offset
    @result_offset
  end

  def result_links
    @result_links
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
