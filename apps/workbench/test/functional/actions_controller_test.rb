require 'test_helper'

class ActionsControllerTest < ActionController::TestCase

  test "send report" do
    post :report_issue, {format: 'js'}, session_for(:admin)
    assert_response :success

    found_email = false
    ActionMailer::Base.deliveries.andand.each do |email|
      if email.subject.include? "Issue reported by admin"
        found_email = true
        break
      end
    end
    assert_equal true, found_email, 'Expected email after issue reported'
  end

end
