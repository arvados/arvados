class PipelineInstanceState < ActiveRecord::Migration
  include CurrentApiClient

  def up
    add_column :pipeline_instances, :state, :string
    add_column :pipeline_instances, :components_summary, :text

    act_as_system_user do
      PipelineInstance.all.each do |pi|
        pi.state = PipelineInstance::New

        if !pi.attribute_present? :success   # success is nil
          if pi[:active] == true
            pi.state = PipelineInstance::RunningOnServer
          else
            if PipelineInstance.is_ready pi.components
              pi.state = PipelineInstance::Ready
            else
              pi.state = PipelineInstance::New
            end
          end
        elsif pi[:success] == true
          pi.state = PipelineInstance::Complete
        else
          pi.state = PipelineInstance::Failed
        end

        pi.save!
      end
    end

    remove_column :pipeline_instances, :active
    remove_column :pipeline_instances, :success
  end

  def down
    add_column :pipeline_instances, :success, :boolean, :null => true
    add_column :pipeline_instances, :active, :boolean, :default => false

    act_as_system_user do
      PipelineInstance.all.each do |pi|
        case pi.state
        when PipelineInstance::New, PipelineInstance::Ready
          pi.active = false
          pi.success = nil
        when PipelineInstance::RunningOnServer
          pi.active = true
          pi.success = nil
        when PipelineInstance::RunningOnClient
          pi.active = false
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

    remove_column :pipeline_instances, :state
  end
end
