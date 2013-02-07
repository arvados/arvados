class OrvosBase < ActiveRecord::Base
  self.abstract_class = true

  def self.columns
    return @columns unless @columns.nil?
    @columns = []
    return @columns if $orvos_api_client.orvos_schema[self.to_s.to_sym].nil?
    $orvos_api_client.orvos_schema[self.to_s.to_sym].each do |coldef|
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
  def self.find(uuid)
    if uuid.class != String or uuid.length < 27 then
      raise 'argument to find() must be a uuid string. Acceptable formats: warehouse locator or string with format xxxxx-xxxxx-xxxxxxxxxxxxxxx'
    end
    new.private_reload(uuid)
  end
  def self.where(*args)
    OrvosResourceList.new(self).where(*args)
  end
  def self.limit(*args)
    OrvosResourceList.new(self).limit(*args)
  end
  def self.eager(*args)
    OrvosResourceList.new(self).eager(*args)
  end
  def self.all(*args)
    OrvosResourceList.new(self).all(*args)
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
      resp = $orvos_api_client.api(self.class, '/' + uuid, postdata)
    else
      resp = $orvos_api_client.api(self.class, '', postdata)
    end
    return false if !resp[:etag] || !resp[:uuid]

    # set read-only non-database attributes
    @etag = resp[:etag]
    @kind = resp[:kind]

    # these attrs can be modified by "save" -- we should update our copies
    %w(uuid owner created_at
       modified_at modified_by_user modified_by_client
      ).each do |attr|
      self.send(attr + '=', resp[attr.to_sym])
    end

    self
  end
  def save!
    self.save or raise Exception.new("Save failed")
  end
  def links(*args)
    o = {}
    o.merge!(args.pop) if args[-1].is_a? Hash
    o[:link_class] ||= args.shift
    o[:name] ||= args.shift
    o[:head_kind] ||= args.shift
    o[:tail_kind] = self.kind
    o[:tail_uuid] = self.uuid
    if all_links
      return all_links.select do |m|
        ok = true
        o.each do |k,v|
          if !v.nil?
            test_v = m.send(k)
            if (v.respond_to?(:uuid) ? v.uuid : v.to_s) != (test_v.respond_to?(:uuid) ? test_v.uuid : test_v.to_s)
              ok = false
            end
          end
        end
        ok
      end
    end
    @links = $orvos_api_client.api Link, '', { _method: 'GET', where: o, eager: true }
    @links = $orvos_api_client.unpack_api_response(@links)
  end
  def all_links
    return @all_links if @all_links
    res = $orvos_api_client.api Link, '', {
      _method: 'GET',
      where: {
        tail_kind: self.kind,
        tail_uuid: self.uuid
      },
      eager: true
    }
    @all_links = $orvos_api_client.unpack_api_response(res)
  end
  def reload
    private_reload(self.uuid)
  end
  def private_reload(uuid_or_hash)
    raise "No such object" if !uuid_or_hash
    if uuid_or_hash.is_a? Hash
      hash = uuid_or_hash
    else
      hash = $orvos_api_client.api(self.class, '/' + uuid_or_hash)
    end
    hash.each do |k,v|
      if self.respond_to?(k.to_s + '=')
        self.send(k.to_s + '=', v)
      else
        # When OrvosApiClient#schema starts telling us what to expect
        # in API responses (not just the server side database
        # columns), this sort of awfulness can be avoided:
        self.instance_variable_set('@' + k.to_s, v)
        if !self.respond_to? k
          singleton = class << self; self end
          singleton.send :define_method, k, lambda { instance_variable_get('@' + k.to_s) }
        end
      end
    end
    @all_links = nil
    self
  end
  def dup
    super.forget_uuid!
  end

  protected

  def forget_uuid!
    self.uuid = nil
    @etag = nil
    self
  end
end
