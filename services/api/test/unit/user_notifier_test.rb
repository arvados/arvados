require 'test_helper'

class UserNotifierTest < ActionMailer::TestCase

  # Send the email, then test that it got queued
  test "account is setup" do
    user = users :active
    email = UserNotifier.account_is_setup user

    assert_not_nil email

    # Test the body of the sent email contains what we expect it to
    assert_equal Rails.configuration.user_notifier_email_from, email.from.first
    assert_equal user.email, email.to.first
    assert_equal 'Welcome to Curoverse - shell account enabled', email.subject
    assert (email.body.to_s.include? 'Your Arvados shell account has been set up'),
        'Expected Your Arvados shell account has been set up in email body'
    assert (email.body.to_s.include? Rails.configuration.workbench_address),
        'Expected workbench url in email body'
  end

end
