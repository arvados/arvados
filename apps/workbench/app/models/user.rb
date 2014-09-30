class User < ArvadosBase
  def initialize(*args)
    super(*args)
    @attribute_sortkey['first_name'] = '050'
    @attribute_sortkey['last_name'] = '051'
  end

  def self.current
    res = arvados_api_client.api self, '/current'
    arvados_api_client.unpack_api_response(res)
  end

  def self.system
    @@arvados_system_user ||= begin
                                res = arvados_api_client.api self, '/system'
                                arvados_api_client.unpack_api_response(res)
                              end
  end

  def full_name
    (self.first_name || "") + " " + (self.last_name || "")
  end

  def activate
    self.private_reload(arvados_api_client.api(self.class,
                                               "/#{self.uuid}/activate",
                                               {}))
  end

  def contents params={}
    Group.contents params.merge(uuid: self.uuid)
  end

  def attributes_for_display
    super.reject { |k,v| %w(owner_uuid default_owner_uuid identity_url prefs).index k }
  end

  def attribute_editable? attr, *args
    (not (self.uuid.andand.match(/000000000000000$/) and self.is_admin)) and super
  end

  def friendly_link_name lookup=nil
    [self.first_name, self.last_name].compact.join ' '
  end

  def unsetup
    self.private_reload(arvados_api_client.api(self.class,
                                               "/#{self.uuid}/unsetup",
                                               {}))
  end

  def self.setup params
    arvados_api_client.api(self, "/setup", params)
  end

  def update_profile params
    self.private_reload(arvados_api_client.api(self.class,
                                               "/#{self.uuid}/profile",
                                               params))
  end

end
