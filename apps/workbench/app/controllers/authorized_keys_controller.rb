class AuthorizedKeysController < ApplicationController
  def new
    super
    @object.authorized_user_uuid = current_user.uuid if current_user
    @object.key_type = 'SSH'
  end

  def create
    @object = AuthorizedKey.new authorized_user_uuid: current_user.uuid, key_type: 'SSH'
    super
  end
end
