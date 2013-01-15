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
        serialize k, coldef[:type].to_sym
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
    thelist = api('')
    thelist[:items].collect { |x| new(x) }
  end
  def self.find(uuid)
    new(api('/' + uuid))
  end
  def save
    postdata = {}
    postdata[self.class.to_s.underscore] =
      Hash[self.class.columns.collect do |col|
             [col.name.to_sym, self.send(col.name.to_sym)]
           end]
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

  protected
  def self.api(action, data=nil, o={})
    dataargs = []
    if !data.nil?
      data.each do |k,v|
        dataargs << '-d'
        if v.is_a? String
          dataargs << "#{k}=#{v}"
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
    JSON.parse json, :symbolize_names => true
  end

  def self.orvos_schema
    $orvos_schema ||= api '', nil, {resource_path: 'schema'}
  end
end
