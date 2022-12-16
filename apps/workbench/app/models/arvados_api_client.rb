# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

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
      @api_response = Oj.strict_load(@api_response_s, :symbol_keys => true)
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

  @@profiling_enabled = Rails.configuration.Workbench.ProfilingEnabled
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

  def api(resources_kind, action, data=nil, tokens={}, include_anon_token=true)

    profile_checkpoint

    if not @api_client
      @client_mtx.synchronize do
        @api_client = HTTPClient.new
        @api_client.ssl_config.timeout = Rails.configuration.Workbench.APIClientConnectTimeout
        @api_client.connect_timeout = Rails.configuration.Workbench.APIClientConnectTimeout
        @api_client.receive_timeout = Rails.configuration.Workbench.APIClientReceiveTimeout
        if Rails.configuration.TLS.Insecure
          @api_client.ssl_config.verify_mode = OpenSSL::SSL::VERIFY_NONE
        else
          # Use system CA certificates
          ["/etc/ssl/certs/ca-certificates.crt",
           "/etc/pki/tls/certs/ca-bundle.crt"]
            .select { |ca_path| File.readable?(ca_path) }
            .each { |ca_path| @api_client.ssl_config.add_trust_ca(ca_path) }
        end
        if Rails.configuration.Workbench.APIResponseCompression
          @api_client.transparent_gzip_decompression = true
        end
      end
    end

    resources_kind = class_kind(resources_kind).pluralize if resources_kind.is_a? Class
    url = "#{self.arvados_v1_base}/#{resources_kind}#{action}"

    # Clean up /arvados/v1/../../discovery/v1 to /discovery/v1
    url.sub! '/arvados/v1/../../', '/'

    anon_tokens = [Rails.configuration.Users.AnonymousUserToken].select { |x| !x.empty? && include_anon_token }

    query = {
      'reader_tokens' => ((tokens[:reader_tokens] ||
                           Thread.current[:reader_tokens] ||
                           []) +
                          anon_tokens).to_json,
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
          query[k] = Oj.dump(v, mode: :compat)
        end
      end
    else
      query["_method"] = "GET"
    end

    if @@profiling_enabled
      query["_profile"] = "true"
    end

    headers = {
      "Accept" => "application/json",
      "Authorization" => "OAuth2 " +
                         (tokens[:arvados_api_token] ||
                          Thread.current[:arvados_api_token] ||
                          ''),
      "X-Request-Id" => Thread.current[:request_id] || '',
    }

    profile_checkpoint { "Prepare request #{query["_method"] or "POST"} #{url} #{query[:uuid]} #{query.inspect[0,256]}" }
    msg = @client_mtx.synchronize do
      begin
        @api_client.post(url, query, headers)
      rescue => exception
        raise NoApiResponseException.new(url, exception)
      end
    end
    profile_checkpoint 'API transaction'
    if @@profiling_enabled
      if msg.headers['X-Runtime']
        Rails.logger.info "API server: #{msg.headers['X-Runtime']} runtime reported"
      end
      Rails.logger.info "Content-Encoding #{msg.headers['Content-Encoding'].inspect}, Content-Length #{msg.headers['Content-Length'].inspect}, actual content size #{msg.content.size}"
    end

    begin
      resp = Oj.strict_load(msg.content, :symbol_keys => true)
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
    if Rails.configuration.testing_override_login_url
      uri = URI(Rails.configuration.testing_override_login_url)
      uri.path = "/login"
      uri.query = URI.encode_www_form(params)
      return uri.to_s
    end

    case
    when Rails.configuration.Login.PAM.Enable,
         Rails.configuration.Login.LDAP.Enable,
         Rails.configuration.Login.Test.Enable

      uri = URI.parse(Rails.configuration.Services.Workbench1.ExternalURL.to_s)
      uri.path = "/users/welcome"
      uri.query = URI.encode_www_form(params)
    else
      uri = URI.parse(Rails.configuration.Services.Controller.ExternalURL.to_s)
      uri.path = "/login"
      uri.query = URI.encode_www_form(params)
    end
    uri.to_s
  end

  def arvados_logout_url(params={})
    uri = URI.parse(Rails.configuration.Services.Controller.ExternalURL.to_s)
    if Rails.configuration.testing_override_login_url
      uri = URI(Rails.configuration.testing_override_login_url)
    end
    uri.path = "/logout"
    uri.query = URI.encode_www_form(params)
    uri.to_s
  end

  def arvados_v1_base
    # workaround Ruby 2.3 bug, can't duplicate URI objects
    # https://github.com/httprb/http/issues/388
    u = URI.parse(Rails.configuration.Services.Controller.ExternalURL.to_s)
    u.path = "/arvados/v1"
    u.to_s
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
