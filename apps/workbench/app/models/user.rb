# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class User < ArvadosBase
  def initialize(*args)
    super(*args)
    @attribute_sortkey['first_name'] = '050'
    @attribute_sortkey['last_name'] = '051'
  end

  def self.current
    res = arvados_api_client.api self, '/current', nil, {}, false
    arvados_api_client.unpack_api_response(res)
  end

  def self.merge new_user_token, direction
    # Merge user accounts.
    #
    # If the direction is "in", the current user is merged into the
    # user represented by new_user_token
    #
    # If the direction is "out", the user represented by new_user_token
    # is merged into the current user.

    if direction == "in"
      user_a = new_user_token
      user_b = Thread.current[:arvados_api_token]
      new_group_name = "Migrated from #{Thread.current[:user].email} (#{Thread.current[:user].uuid})"
    elsif direction == "out"
      user_a = Thread.current[:arvados_api_token]
      user_b = new_user_token
      res = arvados_api_client.api self, '/current', nil, {:arvados_api_token => user_b}, false
      user_b_info = arvados_api_client.unpack_api_response(res)
      new_group_name = "Migrated from #{user_b_info.email} (#{user_b_info.uuid})"
    else
      raise "Invalid merge direction, expected 'in' or 'out'"
    end

    # Create a project owned by user_a to accept everything owned by user_b
    res = arvados_api_client.api Group, nil, {:group => {
                                                :name => new_group_name,
                                                :group_class => "project"},
                                              :ensure_unique_name => true},
                                 {:arvados_api_token => user_a}, false
    target = arvados_api_client.unpack_api_response(res)

    # The merge API merges the "current" user (user_b) into the user
    # represented by "new_user_token" (user_a).
    # After merging, the user_b redirects to user_a.
    res = arvados_api_client.api self, '/merge', {:new_user_token => user_a,
                                                  :new_owner_uuid => target[:uuid],
                                                  :redirect_to_new_user => true},
                                 {:arvados_api_token => user_b}, false
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

  def attribute_editable?(attr, ever=nil)
    (ever or not (self.uuid.andand.match(/000000000000000$/) and
                  self.is_admin)) and super
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

  def deletable?
    false
  end

   def self.creatable?
    current_user and current_user.is_admin
   end
end
