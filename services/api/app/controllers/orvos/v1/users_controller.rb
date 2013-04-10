class Orvos::V1::UsersController < ApplicationController
  def current
    @object = current_user
    show
  end
end
