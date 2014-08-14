class IssueReporter < ActionMailer::Base
  default from: Rails.configuration.report_notifier_email_from
  default to: Rails.configuration.report_notifier_email_to

  def send_report(user, data)
    @user = user
    @data = data
    mail(subject: 'Issue reported')
  end
end
