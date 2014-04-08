class UserNotifier < ActionMailer::Base
  default from: Rails.configuration.user_notifier_email_from

  def account_is_setup(user)
    @user = user
    mail(to: user.email, subject: 'Welcome to Curoverse')
  end
end
