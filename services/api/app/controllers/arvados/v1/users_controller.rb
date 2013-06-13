class Arvados::V1::UsersController < ApplicationController
  def current
    @object = current_user
    show
  end
  def system
    @object = system_user
    show
  end
end
