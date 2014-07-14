class EmptyCollection < ActiveRecord::Migration
  include CurrentApiClient

  def up
    empty_collection
  end

  def down
    act_as_system_user do
      empty_collection.destroy
    end
  end
end
