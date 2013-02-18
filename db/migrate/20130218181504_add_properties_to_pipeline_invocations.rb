class AddPropertiesToPipelineInvocations < ActiveRecord::Migration
  def change
    add_column :pipeline_invocations, :properties, :text
  end
end
