class AuthorizedKeysController < ApplicationController
  def new
    super
    @object.authorized_user = current_user.uuid if current_user
  end
end
