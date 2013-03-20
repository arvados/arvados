class AddDefaultOwnerToApiClientAuthorizations < ActiveRecord::Migration
  def change
    add_column :api_client_authorizations, :default_owner, :string
  end
end
