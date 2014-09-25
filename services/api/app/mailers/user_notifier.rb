class UserNotifier < ActionMailer::Base
  include AbstractController::Callbacks

  default from: Rails.configuration.user_notifier_email_from
  before_filter :load_variables

  def account_is_setup(user)
    @user = user
    mail(to: user.email, subject: 'Welcome to Curoverse')
  end

private
  def load_variables
    if Rails.configuration.respond_to?('workbench_address') and
       not Rails.configuration.workbench_address.nil? and
       not Rails.configuration.workbench_address.empty? then
      @wb_address = Rails.configuration.workbench_address
    else
      @wb_address = '(Unfortunately, config.workbench_address is not set, please contact your site administrator)'
    end
  end

end
