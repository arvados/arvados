class UsersController < ApplicationController
  before_filter :ensure_current_user_is_admin
end
