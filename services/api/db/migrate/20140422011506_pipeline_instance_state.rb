class PipelineInstanceState < ActiveRecord::Migration
  include CurrentApiClient

  def up
    add_column :pipeline_instances, :state, :string
    add_column :pipeline_instances, :components_summary, :text

    PipelineInstance.reset_column_information

    act_as_system_user do
      PipelineInstance.all.each do |pi|
        pi.state = PipelineInstance::New

        if !pi.attribute_present? :success   # success is nil
          if pi[:active] == true
            pi.state = PipelineInstance::RunningOnServer
          else
            if pi.components_look_ready?
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

# We want to perform addition of state, and removal of active and success in two phases. Hence comment these statements out.
=begin
    if column_exists?(:pipeline_instances, :active)
      remove_column :pipeline_instances, :active
    end

    if column_exists?(:pipeline_instances, :success)
      remove_column :pipeline_instances, :success
    end
=end
  end

  def down
# We want to perform addition of state, and removal of active and success in two phases. Hence comment these statements out.
=begin
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
=end

    if column_exists?(:pipeline_instances, :components_summary)
      remove_column :pipeline_instances, :components_summary
    end

    if column_exists?(:pipeline_instances, :state)
      remove_column :pipeline_instances, :state
    end
  end
end
