class ArvadosResourceList
  include ArvadosApiClientHelper
  include Enumerable

  attr_reader :resource_class

  def initialize resource_class=nil
    @resource_class = resource_class
    @fetch_multiple_pages = true
    @arvados_api_token = Thread.current[:arvados_api_token]
    @reader_tokens = Thread.current[:reader_tokens]
  end

  def eager(bool=true)
    @eager = bool
    self
  end

  def limit(max_results)
    if not max_results.nil? and not max_results.is_a? Integer
      raise ArgumentError("argument to limit() must be an Integer or nil")
    end
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
    @results
  end

  def to_ary
    results
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

  def to_hash
    Hash[self.collect { |x| [x.uuid, x] }]
  end

  def empty?
    self.first.nil?
  end

  def items_available
    results
    @items_available
  end

  def result_limit
    results
    @result_limit
  end

  def result_offset
    results
    @result_offset
  end

  # Obsolete method retained during api transition.
  def links_for item_or_uuid, link_class=false
    []
  end

  protected

  def each_page
    api_params = {
      _method: 'GET'
    }
    api_params[:where] = @cond if @cond
    api_params[:eager] = '1' if @eager
    api_params[:select] = @select if @select
    api_params[:order] = @orderby_spec if @orderby_spec
    api_params[:filters] = @filters if @filters


    item_count = 0
    offset = @offset || 0
    @result_limit = nil
    @result_offset = nil

    begin
      api_params[:offset] = offset
      api_params[:limit] = (@limit - item_count) if @limit

      res = arvados_api_client.api(@resource_class, '', api_params,
                                   arvados_api_token: @arvados_api_token,
                                   reader_tokens: @reader_tokens)
      items = arvados_api_client.unpack_api_response res

      @items_available = items.items_available if items.respond_to?(:items_available)
      @result_limit = items.limit if (@fetch_multiple_pages == false) and items.respond_to?(:limit)
      @result_offset = items.offset if (@fetch_multiple_pages == false) and items.respond_to?(:offset)

      break if items.nil? or not items.any?

      item_count += items.size
      if items.respond_to?(:offset)
        offset = items.offset + items.size
      else
        offset = item_count
      end

      yield items

      break if @limit and item_count >= @limit
      break if items.respond_to? :items_available and offset >= items.items_available
    end while @fetch_multiple_pages
    self
  end

end
