# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class GroupsController < ApplicationController
  def index
    @groups = Group.filter [['group_class', '!=', 'project'], ['group_class', '!=', 'filter']]
    @group_uuids = @groups.collect &:uuid
    @links_from = Link.where(link_class: 'permission', tail_uuid: @group_uuids).with_count("none")
    @links_to = Link.where(link_class: 'permission', head_uuid: @group_uuids).with_count("none")
    render_index
  end

  def show
    if @object.group_class == 'project' or @object.group_class == 'filter'
      redirect_to(project_path(@object))
    else
      super
    end
  end
end
