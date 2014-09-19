class AddStateToJob < ActiveRecord::Migration
  def up
    if !column_exists?(:jobs, :state)
      add_column :jobs, :state, :string
    end

    Job.reset_column_information

    act_as_system_user do
      Job.all.each do |job|
        # before_save filter will set state based on job status
        job.save!
      end
    end
  end

  def down
    if column_exists?(:jobs, :state)
      remove_column :jobs, :state
    end
  end
end
