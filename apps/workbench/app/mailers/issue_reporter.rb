class IssueReporter < ActionMailer::Base
  default from: Rails.configuration.report_notifier_email_from
  default to: Rails.configuration.report_notifier_email_to

  def send_report(user, params)
    @user = user
    @params = params
    mail(subject: 'Issue reported')
  end
end
