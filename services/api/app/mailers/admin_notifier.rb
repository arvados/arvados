class AdminNotifier < ActionMailer::Base
  include AbstractController::Callbacks

  default from: Rails.configuration.admin_notifier_email_from
  before_filter :load_variables

  def new_user(user)
    @user = user
    if not Rails.configuration.new_user_notification_recipients.empty? then
      @recipients = Rails.configuration.new_user_notification_recipients
      logger.info "Sending mail to #{@recipients} about new user #{@user.uuid} (#{@user.full_name} <#{@user.email}>)"
      mail(to: @recipients,
           subject: "#{Rails.configuration.email_subject_prefix}New user notification"
          )
    end
  end

  def new_inactive_user(user)
    @user = user
    if not Rails.configuration.new_inactive_user_notification_recipients.empty? then
      @recipients = Rails.configuration.new_inactive_user_notification_recipients
      logger.info "Sending mail to #{@recipients} about new user #{@user.uuid} (#{@user.full_name} <#{@user.email}>)"
      mail(to: @recipients,
           subject: "#{Rails.configuration.email_subject_prefix}New inactive user notification"
          )
    end
  end

private
  def load_variables
    if Rails.configuration.respond_to?('workbench_address') and
       not Rails.configuration.workbench_address.nil? and
       not Rails.configuration.workbench_address.empty? then
      @wb_address = Rails.configuration.workbench_address.sub(/\/$/,'') + '/users'
    else
      @wb_address = ''
    end
  end

end
