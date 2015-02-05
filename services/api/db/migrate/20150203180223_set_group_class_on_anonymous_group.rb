class SetGroupClassOnAnonymousGroup < ActiveRecord::Migration
  include CurrentApiClient
  def up
    act_as_system_user do
      anonymous_group.update_attributes group_class: 'role', name: 'Anonymous users', description: 'Anonymous users'
    end
  end

  def down
    act_as_system_user do
      anonymous_group.update_attributes group_class: nil, name: 'Anonymous group', description: 'Anonymous group'
    end
  end
end
