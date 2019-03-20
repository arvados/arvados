# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AdminNotifier < ActionMailer::Base
  include AbstractController::Callbacks

  default from: Rails.configuration.Users["AdminNotifierEmailFrom"]

  def new_user(user)
    @user = user
    if not Rails.configuration.Users["NewUserNotificationRecipients"].empty? then
      @recipients = Rails.configuration.Users["NewUserNotificationRecipients"]
      logger.info "Sending mail to #{@recipients} about new user #{@user.uuid} (#{@user.full_name} <#{@user.email}>)"

      add_to_subject = ''
      if Rails.configuration.Users["AutoSetupNewUsers"]
        add_to_subject = @user.is_invited ? ' and setup' : ', but not setup'
      end

      mail(to: @recipients,
           subject: "#{Rails.configuration.Users["EmailSubjectPrefix"]}New user created#{add_to_subject} notification"
          )
    end
  end

  def new_inactive_user(user)
    @user = user
    if not Rails.configuration.Users["NewInactiveUserNotificationRecipients"].empty? then
      @recipients = Rails.configuration.Users["NewInactiveUserNotificationRecipients"]
      logger.info "Sending mail to #{@recipients} about new user #{@user.uuid} (#{@user.full_name} <#{@user.email}>)"
      mail(to: @recipients,
           subject: "#{Rails.configuration.Users["EmailSubjectPrefix"]}New inactive user notification"
          )
    end
  end

end
