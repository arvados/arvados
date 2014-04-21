class PipelineInstanceState < ActiveRecord::Migration
  include CurrentApiClient

  def up
    if !column_exists?(:pipeline_instances, :state)
      add_column :pipeline_instances, :state, :string
    end

    if !column_exists?(:pipeline_instances, :components_summary)
      add_column :pipeline_instances, :components_summary, :text
    end

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

    if column_exists?(:pipeline_instances, :active)
      remove_column :pipeline_instances, :active
    end

    if column_exists?(:pipeline_instances, :success)
      remove_column :pipeline_instances, :success
    end
  end

  def down
    if !column_exists?(:pipeline_instances, :success)
      add_column :pipeline_instances, :success, :null => true
    end

    if !column_exists?(:pipeline_instances, :active)
      add_column :pipeline_instances, :active, :default => false
    end

    act_as_system_user do
      PipelineInstance.all.each do |pi|
        if !pi.state
          next
        end

        if pi.state == 'Complete'
          pi.success = true
        elsif pi.state == 'Failed'
          pi.success = false
        elsif pi.state != 'New'
          pi.active = true
        end

        pi.save!
      end
    end

    if column_exists?(:pipeline_instances, :components_summary)
      remove_column :pipeline_instances, :components_summary
    end

    if column_exists?(:pipeline_instances, :state)
      remove_column :pipeline_instances, :state
    end
  end
end
