class ActivateAllAdmins < ActiveRecord::Migration
  def up
    User.update_all({is_active: true}, ['is_admin=?', true])
  end

  def down
  end
end
