# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::UserAgreementsController < ApplicationController
  before_action :admin_required, except: [:index, :sign, :signatures]
  skip_before_action :find_object_by_uuid, only: [:sign, :signatures]
  skip_before_action :render_404_if_no_object, only: [:sign, :signatures]

  def model_class
    Link
  end

  def table_name
    'links'
  end

  def index
    if not current_user.is_invited
      # New users cannot see user agreements until/unless invited to
      # use this installation.
      @objects = []
    else
      act_as_system_user do
        uuids = Link.where("owner_uuid = ? and link_class = ? and name = ? and tail_uuid = ? and head_uuid like ?",
                           system_user_uuid,
                           'signature',
                           'require',
                           system_user_uuid,
                           Collection.uuid_like_pattern).
          collect(&:head_uuid)
        @objects = Collection.where('uuid in (?)', uuids)
      end
    end
    @response_resource_name = 'collection'
    super
  end

  def self._signatures_method_description
    "List all user agreement signature links from a user."
  end

  def signatures
    current_user_uuid = (current_user.andand.is_admin && params[:uuid]) ||
      current_user.uuid
    act_as_system_user do
      @objects = Link.where("owner_uuid = ? and link_class = ? and name = ? and tail_uuid = ? and head_uuid like ?",
                            system_user_uuid,
                            'signature',
                            'click',
                            current_user_uuid,
                            Collection.uuid_like_pattern)
    end
    @response_resource_name = 'link'
    render_list
  end

  def self._sign_method_description
    "Create a signature link from the current user for a given user agreement."
  end

  def sign
    current_user_uuid = current_user.uuid
    act_as_system_user do
      @object = Link.create(link_class: 'signature',
                            name: 'click',
                            tail_uuid: current_user_uuid,
                            head_uuid: params[:uuid])
    end
    show
  end

  def create
    usage_error
  end

  def update
    usage_error
  end

  def destroy
    usage_error
  end

  protected
  def usage_error
    raise ArgumentError.new \
    "Manage user agreements via Collections and Links instead."
  end
  
end
