class IssueReporter < ActionMailer::Base
  default from: Rails.configuration.issue_reporter_email_from
  default to: Rails.configuration.issue_reporter_email_to

  def send_report(user, params)
    @user = user
    @params = params
    subject = 'Issue reported'
    subject += " by #{@user.email}" if @user
    mail(subject: subject)
  end
end
