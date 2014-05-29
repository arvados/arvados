class AddDescriptionToPipelineTemplates < ActiveRecord::Migration
  def change
    add_column :pipeline_templates, :description, :text
  end
end
