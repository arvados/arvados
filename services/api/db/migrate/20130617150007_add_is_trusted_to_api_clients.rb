class AddIsTrustedToApiClients < ActiveRecord::Migration
  def change
    add_column :api_clients, :is_trusted, :boolean, :default => false
  end
end
