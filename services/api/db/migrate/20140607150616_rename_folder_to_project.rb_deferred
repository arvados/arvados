class RenameFolderToProject < ActiveRecord::Migration
  def up
    Group.update_all("group_class = 'project'", "group_class = 'folder'")
  end

  def down
    Group.update_all("group_class = 'folder'", "group_class = 'project'")
  end
end
