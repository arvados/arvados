class AddStateToJob < ActiveRecord::Migration
  include CurrentApiClient

  def up
    ActiveRecord::Base.transaction do
      add_column :jobs, :state, :string
      Job.reset_column_information
      Job.update_all({state: 'Cancelled'}, ['state is null and cancelled_at is not null'])
      Job.update_all({state: 'Failed'}, ['state is null and success = ?', false])
      Job.update_all({state: 'Complete'}, ['state is null and success = ?', true])
      Job.update_all({state: 'Running'}, ['state is null and running = ?', true])
      # Locked/started, but not Running/Failed/Complete? Let's assume it failed.
      Job.update_all({state: 'Failed'}, ['state is null and (is_locked_by_uuid is not null or started_at is not null)'])
      Job.update_all({state: 'Queued'}, ['state is null'])
    end
  end

  def down
    remove_column :jobs, :state
  end
end
