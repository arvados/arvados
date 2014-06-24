class UserSessionsController < ApplicationController

skip_filter :set_thread_api_token, only: [:logged_out]
skip_filter :require_thread_api_token, only: [:logged_out]
skip_filter :use_anonymous_token_if_necessary, only: [:logged_out]
skip_filter :check_user_agreements, only: [:logged_out]
skip_filter :check_user_notifications, only: [:logged_out]
skip_filter :find_object_by_uuid, only: [:logged_out]

#skip_filter _process_action_callbacks.map(&:logged_out)

  def logged_out
  end

end
