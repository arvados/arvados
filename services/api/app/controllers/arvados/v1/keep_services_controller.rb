# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::KeepServicesController < ApplicationController

  skip_before_action :find_object_by_uuid, only: :accessible
  skip_before_action :render_404_if_no_object, only: :accessible
  skip_before_action :require_auth_scope, only: :accessible

  def find_objects_for_index
    # all users can list all keep services
    @objects = KeepService.all
    super
  end

  def self._accessible_method_description
    "List Keep services that the current client can access."
  end

  def accessible
    if request.headers['X-External-Client'] == '1'
      @objects = KeepService.where('service_type=?', 'proxy')
    else
      @objects = KeepService.where('service_type<>?', 'proxy')
    end
    render_list
  end
end
