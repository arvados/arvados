class RequestShellAccessReporter < ActionMailer::Base
  default from: Rails.configuration.email_from
  default to: Rails.configuration.support_email_address

  def send_request(user, params)
    @user = user
    @params = params
    subject = "Shell account request from #{user.full_name} (#{user.email}, #{user.uuid})"
    mail(subject: subject)
  end
end
