class SerializedColumnsAcceptNull < ActiveRecord::Migration
  def change
    change_column :api_client_authorizations, :scopes, :text, null: true, default: '["all"]'
  end
end
