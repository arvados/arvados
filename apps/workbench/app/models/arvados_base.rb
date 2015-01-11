class ArvadosBase < ActiveRecord::Base
  self.abstract_class = true
  attr_accessor :attribute_sortkey
  attr_accessor :create_params

  def self.arvados_api_client
    ArvadosApiClient.new_or_current
  end

  def arvados_api_client
    ArvadosApiClient.new_or_current
  end

  def self.uuid_infix_object_kind
    @@uuid_infix_object_kind ||=
      begin
        infix_kind = {}
        arvados_api_client.discovery[:schemas].each do |name, schema|
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

  def initialize raw_params={}, create_params={}
    super self.class.permit_attribute_params(raw_params)
    @create_params = create_params
    @attribute_sortkey ||= {
      'id' => nil,
      'name' => '000',
      'owner_uuid' => '002',
      'event_type' => '100',
      'link_class' => '100',
      'group_class' => '100',
      'tail_uuid' => '101',
      'head_uuid' => '102',
      'object_uuid' => '102',
      'summary' => '104',
      'description' => '104',
      'properties' => '150',
      'info' => '150',
      'created_at' => '200',
      'modified_at' => '201',
      'modified_by_user_uuid' => '202',
      'modified_by_client_uuid' => '203',
      'uuid' => '999',
    }
  end

  def self.columns
    return @columns if @columns.andand.any?
    @columns = []
    @attribute_info ||= {}
    schema = arvados_api_client.discovery[:schemas][self.to_s.to_sym]
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

    if self == ArvadosBase
      # Determine type from uuid and defer to the appropriate subclass.
      return resource_class_for_uuid(uuid).find(uuid, opts)
    end

    # Only do one lookup on the API side per {class, uuid, workbench
    # request} unless {cache: false} is given via opts.
    cache_key = "request_#{Thread.current.object_id}_#{self.to_s}_#{uuid}"
    if opts[:cache] == false
      Rails.cache.write cache_key, arvados_api_client.api(self, '/' + uuid)
    end
    hash = Rails.cache.fetch cache_key do
      arvados_api_client.api(self, '/' + uuid)
    end
    new.private_reload(hash)
  end

  def self.find?(*args)
    find(*args) rescue nil
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

  def self.select(*args)
    ArvadosResourceList.new(self).select(*args)
  end

  def self.eager(*args)
    ArvadosResourceList.new(self).eager(*args)
  end

  def self.all
    ArvadosResourceList.new(self)
  end

  def self.permit_attribute_params raw_params
    # strong_parameters does not provide security in Workbench: anyone
    # who can get this far can just as well do a call directly to our
    # database (Arvados) with the same credentials we use.
    #
    # The following permit! is necessary even with
    # "ActionController::Parameters.permit_all_parameters = true",
    # because permit_all does not permit nested attributes.
    ActionController::Parameters.new(raw_params).permit!
  end

  def self.create raw_params={}, create_params={}
    x = super(permit_attribute_params(raw_params))
    x.create_params = create_params
    x
  end

  def update_attributes raw_params={}
    super(self.class.permit_attribute_params(raw_params))
  end

  def save
    obdata = {}
    self.class.columns.each do |col|
      unless self.send(col.name.to_sym).nil? and !self.changed.include?(col.name)
          obdata[col.name.to_sym] = self.send(col.name.to_sym)
      end
    end
    obdata.delete :id
    postdata = { self.class.to_s.underscore => obdata }
    if etag
      postdata['_method'] = 'PUT'
      obdata.delete :uuid
      resp = arvados_api_client.api(self.class, '/' + uuid, postdata)
    else
      postdata.merge!(@create_params) if @create_params
      resp = arvados_api_client.api(self.class, '', postdata)
    end
    return false if !resp[:etag] || !resp[:uuid]

    # set read-only non-database attributes
    @etag = resp[:etag]
    @kind = resp[:kind]

    # attributes can be modified during "save" -- we should update our copies
    resp.keys.each do |attr|
      if self.respond_to? "#{attr}=".to_sym
        self.send(attr.to_s + '=', resp[attr.to_sym])
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
      resp = arvados_api_client.api(self.class, '/' + uuid, postdata)
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
    @links = arvados_api_client.api Link, '', { _method: 'GET', where: o, eager: true }
    @links = arvados_api_client.unpack_api_response(@links)
  end

  def all_links
    return @all_links if @all_links
    res = arvados_api_client.api Link, '', {
      _method: 'GET',
      where: {
        tail_kind: self.kind,
        tail_uuid: self.uuid
      },
      eager: true
    }
    @all_links = arvados_api_client.unpack_api_response(res)
  end

  def reload
    private_reload(self.uuid)
  end

  def private_reload(uuid_or_hash)
    raise "No such object" if !uuid_or_hash
    if uuid_or_hash.is_a? Hash
      hash = uuid_or_hash
    else
      hash = arvados_api_client.api(self.class, '/' + uuid_or_hash)
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

  def initialize_copy orig
    super
    forget_uuid!
  end

  def attributes_for_display
    self.attributes.reject { |k,v|
      attribute_sortkey.has_key?(k) and !attribute_sortkey[k]
    }.sort_by { |k,v|
      attribute_sortkey[k] or k
    }
  end

  def class_for_display
    self.class.to_s.underscore.humanize
  end

  def self.class_for_display
    self.to_s.underscore.humanize
  end

  # Array of strings that are names of attributes that should be rendered as textile.
  def textile_attributes
    []
  end

  def self.creatable?
    current_user
  end

  def self.goes_in_projects?
    false
  end

  # can this class of object be copied into a project?
  # override to false on indivudal model classes for which this should not be true
  def self.copies_to_projects?
    self.goes_in_projects?
  end

  def editable?
    (current_user and current_user.is_active and
     (current_user.is_admin or
      current_user.uuid == self.owner_uuid or
      new_record? or
      (respond_to?(:writable_by) ?
       writable_by.include?(current_user.uuid) :
       (ArvadosBase.find(owner_uuid).writable_by.include? current_user.uuid rescue false)))) or false
  end

  # Array of strings that are the names of attributes that can be edited
  # with X-Editable.
  def editable_attributes
    self.class.columns.map(&:name) -
      %w(created_at modified_at modified_by_user_uuid modified_by_client_uuid updated_at)
  end

  def attribute_editable?(attr, ever=nil)
    if not editable_attributes.include?(attr.to_s)
      false
    elsif not (current_user.andand.is_active)
      false
    elsif attr == 'uuid'
      current_user.is_admin
    elsif ever
      true
    else
      editable?
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
      resource_class ||= arvados_api_client.
        kind_class(self.uuid_infix_object_kind[re[1]])
    end
    if opts[:referring_object] and
        opts[:referring_attr] and
        opts[:referring_attr].match /_uuid$/
      resource_class ||= arvados_api_client.
        kind_class(opts[:referring_object].
                   attributes[opts[:referring_attr].
                              sub(/_uuid$/, '_kind')])
    end
    resource_class
  end

  def resource_param_name
    self.class.to_s.underscore
  end

  def friendly_link_name lookup=nil
    (name if self.respond_to? :name) || default_name
  end

  def content_summary
    self.class_for_display
  end

  def selection_label
    friendly_link_name
  end

  def self.default_name
    self.to_s.underscore.humanize
  end

  def controller
    (self.class.to_s.pluralize + 'Controller').constantize
  end

  def controller_name
    self.class.to_s.tableize
  end

  # Placeholder for name when name is missing or empty
  def default_name
    if self.respond_to? :name
      "New #{class_for_display.downcase}"
    else
      uuid
    end
  end

  def owner
    ArvadosBase.find(owner_uuid) rescue nil
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
