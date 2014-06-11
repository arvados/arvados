class AddDockerLocatorToJobs < ActiveRecord::Migration
  def change
    add_column :jobs, :docker_image_locator, :string
  end
end
