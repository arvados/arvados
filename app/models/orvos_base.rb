class OrvosBase < ActiveRecord::Base
  @@orvos_v1_base = Rails.configuration.orvos_v1_base
  def self.columns
    return @columns unless @columns.nil?
    @columns = []
    return @columns if orvos_schema[self.to_s.to_sym].nil?
    orvos_schema[self.to_s.to_sym].each do |coldef|
      k = coldef[:name].to_sym
      if coldef[:type] == coldef[:type].downcase
        @columns << column(k, coldef[:type].to_sym)
      else
        @columns << column(k, :text)
        serialize k, coldef[:type].constantize
      end
      attr_accessible k
    end
    attr_reader :etag
    attr_reader :kind
    @columns
  end
  def self.column(name, sql_type = nil, default = nil, null = true)
    ActiveRecord::ConnectionAdapters::Column.new(name.to_s, default, sql_type.to_s, null)
  end
  def self.all
    unpack_api_response(api(''))
  end
  def self.find(uuid)
    new(api('/' + uuid))
  end
  def self.where(cond)
    all.select do |o|
      0 == cond.select do |k,v|
        o.send(k) != v
      end.size
    end
  end
  def save
    obdata = {}
    self.class.columns.each do |col|
      obdata[col.name.to_sym] = self.send(col.name.to_sym)
    end
    obdata.delete :id
    obdata.delete :uuid
    postdata = { self.class.to_s.underscore => obdata }
    if etag
      postdata['_method'] = 'PUT'
      resp = self.class.api('/' + uuid, postdata)
    else
      resp = self.class.api('', postdata)
    end
    return false if !resp[:etag] || !resp[:uuid]
    @etag = resp[:etag]
    @kind = resp[:kind]
    self.uuid ||= resp[:uuid]
    self
  end
  def save!
    self.save or raise Exception.new("Save failed")
  end
  def initialize(h={})
    @etag = h.delete :etag
    @kind = h.delete :kind
    super
  end
  def metadata(*args)
    o = {}
    o.merge!(args.pop) if args[-1].is_a? Hash
    o[:metadata_class] ||= args.shift
    o[:name] ||= args.shift
    o[:head_kind] ||= args.shift
    o[:tail_kind] = self.kind
    o[:tail] = self.uuid
    @metadata = self.class.api '', { _method: 'GET', where: o, eager: true }, { resource_path: 'metadata' }
    @metadata = self.class.unpack_api_response(@metadata)
  end

  protected
  def self.api(action, data=nil, o={})
    dataargs = []
    if !data.nil?
      data.each do |k,v|
        dataargs << '-d'
        if v.is_a? String or v.nil?
          dataargs << "#{k}=#{v}"
        elsif v == true or v == false
          dataargs << "#{k}=#{v ? 1 : 0}"
        else
          dataargs << "#{k}=#{JSON.generate v}"
        end
      end
    end
    json = nil
    IO.popen([ENV,
              'curl',
              '-sk',
              *dataargs,
              "#{@@orvos_v1_base}/#{o[:resource_path] || self.to_s.underscore.pluralize}#{action}"],
             'r') do |io|
      json = io.read
    end
    resp = JSON.parse json, :symbolize_names => true
    if resp[:errors]
      raise "API errors:\n#{resp[:errors].join "\n"}\n"
    end
    resp
  end

  def self.orvos_schema
    $orvos_schema ||= api '', nil, {resource_path: 'schema'}
  end

  def self.kind_class(kind)
    kind.match(/^orvos\#(.+?)(_list|List)?$/)[1].pluralize.classify.constantize rescue nil
  end

  def self.unpack_api_response(j, kind=nil)
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
        oclass.new(j)
      else
        j
      end
    else
      j
    end
  end
end
