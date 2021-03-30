# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Group < ArvadosBase
  def self.goes_in_projects?
    true
  end

  def self.copies_to_projects?
    false
  end

  def self.contents params={}
    res = arvados_api_client.api self, "/contents", {
      _method: 'GET'
    }.merge(params)
    ret = ArvadosResourceList.new
    ret.results = arvados_api_client.unpack_api_response(res)
    ret
  end

  def editable?
    if group_class == 'filter'
      return false
    end
    super
  end

  def contents params={}
    res = arvados_api_client.api self.class, "/#{self.uuid}/contents", {
      _method: 'GET'
    }.merge(params)
    ret = ArvadosResourceList.new
    ret.results = arvados_api_client.unpack_api_response(res)
    ret
  end

  def class_for_display
    (group_class == 'project' or group_class == 'filter') ? 'Project' : super
  end

  def textile_attributes
    [ 'description' ]
  end

  def self.creatable?
    false
  end

  def untrash
    arvados_api_client.api(self.class, "/#{self.uuid}/untrash", {"ensure_unique_name" => true})
  end
end
