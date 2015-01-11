require 'httpclient'
require 'thread'

class ArvadosApiClient
  class ApiError < StandardError
    attr_reader :api_response, :api_response_s, :api_status, :request_url

    def initialize(request_url, errmsg)
      @request_url = request_url
      @api_response ||= {}
      errors = @api_response[:errors]
      if not errors.is_a?(Array)
        @api_response[:errors] = [errors || errmsg]
      end
      super(errmsg)
    end
  end

  class NoApiResponseException < ApiError
    def initialize(request_url, exception)
      @api_response_s = exception.to_s
      super(request_url,
            "#{exception.class.to_s} error connecting to API server")
    end
  end

  class InvalidApiResponseException < ApiError
    def initialize(request_url, api_response)
      @api_status = api_response.status_code
      @api_response_s = api_response.content
      super(request_url, "Unparseable response from API server")
    end
  end

  class ApiErrorResponseException < ApiError
    def initialize(request_url, api_response)
      @api_status = api_response.status_code
      @api_response_s = api_response.content
      @api_response = Oj.load(@api_response_s, :symbol_keys => true)
      errors = @api_response[:errors]
      if errors.respond_to?(:join)
        errors = errors.join("\n\n")
      else
        errors = errors.to_s
      end
      super(request_url, "#{errors} [API: #{@api_status}]")
    end
  end

  class AccessForbiddenException < ApiErrorResponseException; end
  class NotFoundException < ApiErrorResponseException; end
  class NotLoggedInException < ApiErrorResponseException; end

  ERROR_CODE_CLASSES = {
    401 => NotLoggedInException,
    403 => AccessForbiddenException,
    404 => NotFoundException,
  }

  @@profiling_enabled = Rails.configuration.profiling_enabled
  @@discovery = nil

  # An API client object suitable for handling API requests on behalf
  # of the current thread.
  def self.new_or_current
    # If this thread doesn't have an API client yet, *or* this model
    # has been reloaded since the existing client was created, create
    # a new client. Otherwise, keep using the latest client created in
    # the current thread.
    unless Thread.current[:arvados_api_client].andand.class == self
      Thread.current[:arvados_api_client] = new
    end
    Thread.current[:arvados_api_client]
  end

  def initialize *args
    @api_client = nil
    @client_mtx = Mutex.new
  end

  def api(resources_kind, action, data=nil, tokens={})

    profile_checkpoint

    if not @api_client
      @client_mtx.synchronize do
        @api_client = HTTPClient.new
        if Rails.configuration.arvados_insecure_https
          @api_client.ssl_config.verify_mode = OpenSSL::SSL::VERIFY_NONE
        else
          # Use system CA certificates
          @api_client.ssl_config.add_trust_ca('/etc/ssl/certs')
        end
      end
    end

    resources_kind = class_kind(resources_kind).pluralize if resources_kind.is_a? Class
    url = "#{self.arvados_v1_base}/#{resources_kind}#{action}"

    # Clean up /arvados/v1/../../discovery/v1 to /discovery/v1
    url.sub! '/arvados/v1/../../', '/'

    query = {
      'api_token' => tokens[:arvados_api_token] || Thread.current[:arvados_api_token] || '',
      'reader_tokens' => (tokens[:reader_tokens] || Thread.current[:reader_tokens] || []).to_json,
    }
    if !data.nil?
      data.each do |k,v|
        if v.is_a? String or v.nil?
          query[k] = v
        elsif v == true
          query[k] = 1
        elsif v == false
          query[k] = 0
        else
          query[k] = JSON.dump(v)
        end
      end
    else
      query["_method"] = "GET"
    end
    if @@profiling_enabled
      query["_profile"] = "true"
    end

    header = {"Accept" => "application/json"}

    profile_checkpoint { "Prepare request #{url} #{query[:uuid]} #{query[:where]} #{query[:filters]} #{query[:order]}" }
    msg = @client_mtx.synchronize do
      begin
        @api_client.post(url, query, header: header)
      rescue => exception
        raise NoApiResponseException.new(url, exception)
      end
    end
    profile_checkpoint 'API transaction'

    begin
      resp = Oj.load(msg.content, :symbol_keys => true)
    rescue Oj::ParseError
      resp = nil
    end
    if not resp.is_a? Hash
      raise InvalidApiResponseException.new(url, msg)
    elsif msg.status_code != 200
      error_class = ERROR_CODE_CLASSES.fetch(msg.status_code,
                                             ApiErrorResponseException)
      raise error_class.new(url, msg)
    end

    if resp[:_profile]
      Rails.logger.info "API client: " \
      "#{resp.delete(:_profile)[:request_time]} request_time"
    end
    profile_checkpoint 'Parse response'
    resp
  end

  def self.patch_paging_vars(ary, items_available, offset, limit, links=nil)
    if items_available
      (class << ary; self; end).class_eval { attr_accessor :items_available }
      ary.items_available = items_available
    end
    if offset
      (class << ary; self; end).class_eval { attr_accessor :offset }
      ary.offset = offset
    end
    if limit
      (class << ary; self; end).class_eval { attr_accessor :limit }
      ary.limit = limit
    end
    if links
      (class << ary; self; end).class_eval { attr_accessor :links }
      ary.links = links
    end
    ary
  end

  def unpack_api_response(j, kind=nil)
    if j.is_a? Hash and j[:items].is_a? Array and j[:kind].match(/(_list|List)$/)
      ary = j[:items].collect { |x| unpack_api_response x, x[:kind] }
      links = ArvadosResourceList.new Link
      links.results = (j[:links] || []).collect do |x|
        unpack_api_response x, x[:kind]
      end
      self.class.patch_paging_vars(ary, j[:items_available], j[:offset], j[:limit], links)
    elsif j.is_a? Hash and (kind || j[:kind])
      oclass = self.kind_class(kind || j[:kind])
      if oclass
        j.keys.each do |k|
          childkind = j["#{k.to_s}_kind".to_sym]
          if childkind
            j[k] = self.unpack_api_response(j[k], childkind)
          end
        end
        oclass.new.private_reload(j)
      else
        j
      end
    else
      j
    end
  end

  def arvados_login_url(params={})
    if Rails.configuration.respond_to? :arvados_login_base
      uri = Rails.configuration.arvados_login_base
    else
      uri = self.arvados_v1_base.sub(%r{/arvados/v\d+.*}, '/login')
    end
    if params.size > 0
      uri += '?' << params.collect { |k,v|
        CGI.escape(k.to_s) + '=' + CGI.escape(v.to_s)
      }.join('&')
    end
    uri
  end

  def arvados_logout_url(params={})
    arvados_login_url(params).sub('/login','/logout')
  end

  def arvados_v1_base
    Rails.configuration.arvados_v1_base
  end

  def discovery
    @@discovery ||= api '../../discovery/v1/apis/arvados/v1/rest', ''
  end

  def kind_class(kind)
    kind.match(/^arvados\#(.+?)(_list|List)?$/)[1].pluralize.classify.constantize rescue nil
  end

  def class_kind(resource_class)
    resource_class.to_s.underscore
  end

  def self.class_kind(resource_class)
    resource_class.to_s.underscore
  end

  protected
  def profile_checkpoint label=nil
    return if !@@profiling_enabled
    label = yield if block_given?
    t = Time.now
    if label and @profile_t0
      Rails.logger.info "API client: #{t - @profile_t0} #{label}"
    end
    @profile_t0 = t
  end
end
