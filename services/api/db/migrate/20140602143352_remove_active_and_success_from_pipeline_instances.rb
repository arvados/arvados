# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RemoveActiveAndSuccessFromPipelineInstances < ActiveRecord::Migration
  include CurrentApiClient

  def up
    if column_exists?(:pipeline_instances, :active)
      remove_column :pipeline_instances, :active
    end

    if column_exists?(:pipeline_instances, :success)
      remove_column :pipeline_instances, :success
    end
  end

  def down
    if !column_exists?(:pipeline_instances, :success)
      add_column :pipeline_instances, :success, :boolean, :null => true
    end
    if !column_exists?(:pipeline_instances, :active)
      add_column :pipeline_instances, :active, :boolean, :default => false
    end

    act_as_system_user do
      PipelineInstance.all.each do |pi|
        case pi.state
        when PipelineInstance::New, PipelineInstance::Ready, PipelineInstance::Paused, PipelineInstance::RunningOnClient
          pi.active = nil
          pi.success = nil
        when PipelineInstance::RunningOnServer
          pi.active = true
          pi.success = nil
        when PipelineInstance::Failed
          pi.active = false
          pi.success = false
        when PipelineInstance::Complete
          pi.active = false
          pi.success = true
        end
        pi.save!
      end
    end
  end
end
