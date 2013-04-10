class ApiClientAuthorizationsController < ApplicationController
  def index
    if Thread.current[:api_client_trusted]
      @objects = model_class.
        joins(:user, :api_client).
        where('user_id=?', current_user.id)
    else
      @objects = model_class.where('1=0')
    end
  end
end
