class UserSessionsController < ApplicationController

skip_filter :permit_anonymous_browsing_if_no_thread_token, only: [:logged_out]
skip_filter :set_thread_api_token, only: [:logged_out]
skip_filter :require_thread_api_token, only: [:logged_out]
skip_filter :permit_anonymous_browsing_for_inactive_user, only: [:logged_out]
skip_filter :check_user_agreements, only: [:logged_out]
skip_filter :check_user_notifications, only: [:logged_out]
skip_filter :find_object_by_uuid, only: [:logged_out]

#skip_filter _process_action_callbacks.map(&:logged_out)

  def logged_out
  end

end
