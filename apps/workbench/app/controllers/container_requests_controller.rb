class ContainerRequestsController < ApplicationController
  skip_around_filter :require_thread_api_token, if: proc { |ctrl|
    Rails.configuration.anonymous_user_token and
    'show' == ctrl.action_name
  }

  def show_pane_list
    %w(Status Log Advanced)
  end

  def cancel
    @object.update_attributes! priority: 0
    if params[:return_to]
      redirect_to params[:return_to]
    else
      redirect_to @object
    end
  end
end
