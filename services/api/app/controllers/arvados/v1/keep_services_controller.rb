# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::KeepServicesController < ApplicationController

  skip_before_action :find_object_by_uuid, only: :accessible
  skip_before_action :render_404_if_no_object, only: :accessible
  skip_before_action :require_auth_scope, only: :accessible

  def find_objects_for_index
    # all users can list all keep services
    @objects = from_config_or_db
    super
  end

  def accessible
    if request.headers['X-External-Client'] == '1'
      @objects = from_config_or_db.where('service_type=?', 'proxy')
    else
      @objects = from_config_or_db.where('service_type<>?', 'proxy')
    end
    render_list
  end

  private

  # return the set of keep services from the database (if this is an
  # older installation or test system where entries have been added
  # manually) or, preferably, the cluster config file.
  def from_config_or_db
    if KeepService.all.count == 0
      KeepService.from_config
    else
      KeepService.all
    end
  end
end
