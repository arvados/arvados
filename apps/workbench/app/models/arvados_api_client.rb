require 'httpclient'
require 'thread'

class ArvadosApiClient
  class NotLoggedInException < StandardError
  end
  class InvalidApiResponseException < StandardError
  end

  @@profiling_enabled = Rails.configuration.profiling_enabled
  @@discovery = nil

  # An API client object suitable for handling API requests on behalf
  # of the current thread.
  def self.new_or_current
    Thread.current[:arvados_api_client] ||= new
  end

  def initialize *args
    @api_client = nil
    @client_mtx = Mutex.new
  end

  def api(resources_kind, action, data=nil)
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

    api_token = Thread.current[:arvados_api_token]
    api_token ||= ''

    resources_kind = class_kind(resources_kind).pluralize if resources_kind.is_a? Class
    url = "#{self.arvados_v1_base}/#{resources_kind}#{action}"

    # Clean up /arvados/v1/../../discovery/v1 to /discovery/v1
    url.sub! '/arvados/v1/../../', '/'

    query = {"api_token" => api_token}
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

    profile_checkpoint { "Prepare request #{url} #{query[:uuid]} #{query[:where]}" }
    msg = @client_mtx.synchronize do
      @api_client.post(url, 
                       query,
                       header: header)
    end
    profile_checkpoint 'API transaction'

    if msg.status_code == 401
      raise NotLoggedInException.new
    end

    json = msg.content
    
    begin
      resp = Oj.load(json, :symbol_keys => true)
    rescue Oj::ParseError
      raise InvalidApiResponseException.new json
    end
    if not resp.is_a? Hash
      raise InvalidApiResponseException.new json
    end
    if msg.status_code != 200
      errors = resp[:errors]
      errors = errors.join("\n\n") if errors.is_a? Array
      raise "#{errors} [API: #{msg.status_code}]"
    end
    if resp[:_profile]
      Rails.logger.info "API client: " \
      "#{resp.delete(:_profile)[:request_time]} request_time"
    end
    profile_checkpoint 'Parse response'
    resp
  end

  def self.patch_paging_vars(ary, items_available, offset, limit)
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
    ary
  end

  def unpack_api_response(j, kind=nil)
    if j.is_a? Hash and j[:items].is_a? Array and j[:kind].match(/(_list|List)$/)
      ary = j[:items].collect { |x| unpack_api_response x, x[:kind] }
      self.class.patch_paging_vars(ary, j[:items_available], j[:offset], j[:limit])
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
