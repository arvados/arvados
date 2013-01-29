class OrvosApiClient
  class NotLoggedInException < Exception
  end
  def api(resources_kind, action, data=nil)
    orvos_api_token = Thread.current[:orvos_api_token]
    orvos_api_token = '' if orvos_api_token.nil?
    dataargs = ['--data-urlencode',
                "api_token=#{orvos_api_token}",
                '--header',
                'Accept:application/json']
    if !data.nil?
      data.each do |k,v|
        dataargs << '--data-urlencode'
        if v.is_a? String or v.nil?
          dataargs << "#{k}=#{v}"
        elsif v == true or v == false
          dataargs << "#{k}=#{v ? 1 : 0}"
        else
          dataargs << "#{k}=#{JSON.generate v}"
        end
      end
    else
      dataargs << '--data-urlencode' << '_method=GET'
    end
    json = nil
    resources_kind = class_kind(resources_kind).pluralize if resources_kind.is_a? Class
    url = "#{self.orvos_v1_base}/#{resources_kind}#{action}"
    IO.popen([ENV,
              'curl',
              '-s',
              *dataargs,
              url],
             'r') do |io|
      json = io.read
    end
    resp = JSON.parse json, :symbolize_names => true
    if resp[:errors]
      if resp[:errors][0] == 'Not logged in'
        raise NotLoggedInException.new
      else
        errors = resp[:errors]
        errors = errors.join("\n\n") if errors.is_a? Array
        raise "API errors:\n\n#{errors}\n"
      end
    end
    resp
  end

  def unpack_api_response(j, kind=nil)
    if j.is_a? Hash and j[:items].is_a? Array and j[:kind].match(/(_list|List)$/)
      j[:items].collect { |x| unpack_api_response x, j[:kind] }
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

  def orvos_login_url(params={})
    uri = self.orvos_v1_base.sub(%r{/orvos/v\d+.*}, '/login')
    if params.size > 0
      uri << '?' << params.collect { |k,v|
        CGI.escape(k.to_s) + '=' + CGI.escape(v.to_s)
      }.join('&')
    end
  end

  def orvos_v1_base
    Rails.configuration.orvos_v1_base
  end

  def orvos_schema
    @orvos_schema ||= api 'schema', ''
  end

  def kind_class(kind)
    kind.match(/^orvos\#(.+?)(_list|List)?$/)[1].pluralize.classify.constantize rescue nil
  end

  def class_kind(resource_class)
    resource_class.to_s.underscore
  end
end
