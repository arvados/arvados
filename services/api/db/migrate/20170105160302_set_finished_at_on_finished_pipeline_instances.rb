class SetFinishedAtOnFinishedPipelineInstances < ActiveRecord::Migration
  def change
    ActiveRecord::Base.connection.execute("update pipeline_instances set finished_at=updated_at where finished_at is null and (state='Failed' or state='Complete')")
  end
end
