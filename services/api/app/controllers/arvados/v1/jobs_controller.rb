# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::JobsController < ApplicationController
  accept_attribute_as_json :components, Hash
  accept_attribute_as_json :script_parameters, Hash
  accept_attribute_as_json :runtime_constraints, Hash
  accept_attribute_as_json :tasks_summary, Hash
  skip_before_action :find_object_by_uuid, :only => [:queue, :queue_size]
  skip_before_action :render_404_if_no_object, :only => [:queue, :queue_size]

  include DbCurrentTime

  def create
    return send_error("Unsupported legacy jobs API",
                      status: 400)
  end

  def cancel
    return send_error("Unsupported legacy jobs API",
                      status: 400)
  end

  def lock
    return send_error("Unsupported legacy jobs API",
                      status: 400)
  end

  def queue
    @objects = []
    index
  end

  def queue_size
    render :json => {:queue_size => 0}
  end

  def self._create_requires_parameters
    (super rescue {}).
      merge({
              find_or_create: {
                type: 'boolean', required: false, default: false
              },
              filters: {
                type: 'array', required: false
              },
              minimum_script_version: {
                type: 'string', required: false
              },
              exclude_script_versions: {
                type: 'array', required: false
              },
            })
  end

  def self._queue_requires_parameters
    self._index_requires_parameters
  end

  protected

  def load_filters_param
    begin
      super
      attrs = resource_attrs rescue {}
      @filters = Job.load_job_specific_filters attrs, @filters, @read_users
    rescue ArgumentError => error
      send_error(error.message)
      false
    else
      true
    end
  end
end
