class AddUniqueNameIndexToLinks < ActiveRecord::Migration
  def change
    # todo: add "check (link_class is not 'name' or name is not null)"
    add_index :links, [:tail_uuid, :name], where: "link_class='name'", unique: true
  end
end
