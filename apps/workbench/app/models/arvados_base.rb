class ArvadosBase < ActiveRecord::Base
  self.abstract_class = true
  attr_accessor :attribute_sortkey

  def self.uuid_infix_object_kind
    @@uuid_infix_object_kind ||=
      begin
        infix_kind = {}
        $arvados_api_client.discovery[:schemas].each do |name, schema|
          if schema[:uuidPrefix]
            infix_kind[schema[:uuidPrefix]] =
              'arvados#' + name.to_s.camelcase(:lower)
          end
        end

        # Recognize obsolete types.
        infix_kind.
          merge('mxsvm' => 'arvados#pipelineTemplate', # Pipeline
                'uo14g' => 'arvados#pipelineInstance', # PipelineInvocation
                'ldvyl' => 'arvados#group') # Project
      end
  end

  def initialize(*args)
    super(*args)
    @attribute_sortkey ||= {
      'id' => nil,
      'uuid' => '000',
      'owner_uuid' => '001',
      'created_at' => '002',
      'modified_at' => '003',
      'modified_by_user_uuid' => '004',
      'modified_by_client_uuid' => '005',
      'name' => '050',
      'tail_kind' => '100',
      'tail_uuid' => '100',
      'head_kind' => '101',
      'head_uuid' => '101',
      'info' => 'zzz-000',
      'updated_at' => 'zzz-999'
    }
  end

  def self.columns
    return @columns unless @columns.nil?
    @columns = []
    @attribute_info ||= {}
    schema = $arvados_api_client.discovery[:schemas][self.to_s.to_sym]
    return @columns if schema.nil?
    schema[:properties].each do |k, coldef|
      case k
      when :etag, :kind
        attr_reader k
      else
        if coldef[:type] == coldef[:type].downcase
          # boolean, integer, etc.
          @columns << column(k, coldef[:type].to_sym)
        else
          # Hash, Array
          @columns << column(k, :text)
          serialize k, coldef[:type].constantize
        end
        attr_accessible k
        @attribute_info[k] = coldef
      end
    end
    @columns
  end

  def self.column(name, sql_type = nil, default = nil, null = true)
    ActiveRecord::ConnectionAdapters::Column.new(name.to_s, default, sql_type.to_s, null)
  end

  def self.attribute_info
    self.columns
    @attribute_info
  end

  def self.find(uuid, opts={})
    if uuid.class != String or uuid.length < 27 then
      raise 'argument to find() must be a uuid string. Acceptable formats: warehouse locator or string with format xxxxx-xxxxx-xxxxxxxxxxxxxxx'
    end

    # Only do one lookup on the API side per {class, uuid, workbench
    # request} unless {cache: false} is given via opts.
    cache_key = "request_#{Thread.current.object_id}_#{self.to_s}_#{uuid}"
    if opts[:cache] == false
      Rails.cache.write cache_key, $arvados_api_client.api(self, '/' + uuid)
    end
    hash = Rails.cache.fetch cache_key do
      $arvados_api_client.api(self, '/' + uuid)
    end
    new.private_reload(hash)
  end

  def self.order(*args)
    ArvadosResourceList.new(self).order(*args)
  end

  def self.filter(*args)
    ArvadosResourceList.new(self).filter(*args)
  end

  def self.where(*args)
    ArvadosResourceList.new(self).where(*args)
  end

  def self.limit(*args)
    ArvadosResourceList.new(self).limit(*args)
  end

  def self.eager(*args)
    ArvadosResourceList.new(self).eager(*args)
  end

  def self.all(*args)
    ArvadosResourceList.new(self).all(*args)
  end

  def save
    obdata = {}
    self.class.columns.each do |col|
      obdata[col.name.to_sym] = self.send(col.name.to_sym)
    end
    obdata.delete :id
    postdata = { self.class.to_s.underscore => obdata }
    if etag
      postdata['_method'] = 'PUT'
      obdata.delete :uuid
      resp = $arvados_api_client.api(self.class, '/' + uuid, postdata)
    else
      resp = $arvados_api_client.api(self.class, '', postdata)
    end
    return false if !resp[:etag] || !resp[:uuid]

    # set read-only non-database attributes
    @etag = resp[:etag]
    @kind = resp[:kind]

    # these attrs can be modified by "save" -- we should update our copies
    %w(uuid owner_uuid created_at
       modified_at modified_by_user_uuid modified_by_client_uuid
      ).each do |attr|
      if self.respond_to? "#{attr}=".to_sym
        self.send(attr + '=', resp[attr.to_sym])
      end
    end

    @new_record = false

    self
  end

  def save!
    self.save or raise Exception.new("Save failed")
  end

  def destroy
    if etag || uuid
      postdata = { '_method' => 'DELETE' }
      resp = $arvados_api_client.api(self.class, '/' + uuid, postdata)
      resp[:etag] && resp[:uuid] && resp
    else
      true
    end
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
    @links = $arvados_api_client.api Link, '', { _method: 'GET', where: o, eager: true }
    @links = $arvados_api_client.unpack_api_response(@links)
  end

  def all_links
    return @all_links if @all_links
    res = $arvados_api_client.api Link, '', {
      _method: 'GET',
      where: {
        tail_kind: self.kind,
        tail_uuid: self.uuid
      },
      eager: true
    }
    @all_links = $arvados_api_client.unpack_api_response(res)
  end

  def reload
    private_reload(self.uuid)
  end

  def private_reload(uuid_or_hash)
    raise "No such object" if !uuid_or_hash
    if uuid_or_hash.is_a? Hash
      hash = uuid_or_hash
    else
      hash = $arvados_api_client.api(self.class, '/' + uuid_or_hash)
    end
    hash.each do |k,v|
      if self.respond_to?(k.to_s + '=')
        self.send(k.to_s + '=', v)
      else
        # When ArvadosApiClient#schema starts telling us what to expect
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
    @new_record = false
    self
  end

  def to_param
    uuid
  end

  def dup
    super.forget_uuid!
  end

  def attributes_for_display
    self.attributes.reject { |k,v|
      attribute_sortkey.has_key?(k) and !attribute_sortkey[k]
    }.sort_by { |k,v|
      attribute_sortkey[k] or k
    }
  end

  def self.creatable?
    current_user
  end

  def editable?
    (current_user and current_user.is_active and
     (current_user.is_admin or
      current_user.uuid == self.owner_uuid))
  end

  def attribute_editable?(attr)
    if "created_at modified_at modified_by_user_uuid modified_by_client_uuid updated_at".index(attr.to_s)
      false
    elsif not (current_user.andand.is_active)
      false
    elsif "uuid owner_uuid".index(attr.to_s) or current_user.is_admin
      current_user.is_admin
    else
      current_user.uuid == self.owner_uuid or current_user.uuid == self.uuid
    end
  end

  def self.resource_class_for_uuid(uuid, opts={})
    if uuid.is_a? ArvadosBase
      return uuid.class
    end
    unless uuid.is_a? String
      return nil
    end
    if opts[:class].is_a? Class
      return opts[:class]
    end
    if uuid.match /^[0-9a-f]{32}(\+[^,]+)*(,[0-9a-f]{32}(\+[^,]+)*)*$/
      return Collection
    end
    resource_class = nil
    uuid.match /^[0-9a-z]{5}-([0-9a-z]{5})-[0-9a-z]{15}$/ do |re|
      resource_class ||= $arvados_api_client.
        kind_class(self.uuid_infix_object_kind[re[1]])
    end
    if opts[:referring_object] and
        opts[:referring_attr] and
        opts[:referring_attr].match /_uuid$/
      resource_class ||= $arvados_api_client.
        kind_class(opts[:referring_object].
                   attributes[opts[:referring_attr].
                              sub(/_uuid$/, '_kind')])
    end
    resource_class
  end

  def friendly_link_name
    (name if self.respond_to? :name) || uuid
  end

  def selection_label
    friendly_link_name
  end

  protected

  def forget_uuid!
    self.uuid = nil
    @etag = nil
    self
  end

  def self.current_user
    Thread.current[:user] ||= User.current if Thread.current[:arvados_api_token]
    Thread.current[:user]
  end
  def current_user
    self.class.current_user
  end
end
