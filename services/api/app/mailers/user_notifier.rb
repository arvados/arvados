# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class UserNotifier < ActionMailer::Base
  include AbstractController::Callbacks

  default from: Rails.configuration.Users.UserNotifierEmailFrom

  def account_is_setup(user)
    @user = user
    if not Rails.configuration.Users.UserNotifierEmailBcc.empty? then
      @bcc = Rails.configuration.Users.UserNotifierEmailBcc.keys
      mail(to: user.email, subject: 'Welcome to Arvados - account enabled', bcc: @bcc)
    else
      mail(to: user.email, subject: 'Welcome to Arvados - account enabled')
    end
  end

end
