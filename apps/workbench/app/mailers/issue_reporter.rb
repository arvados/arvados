# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class IssueReporter < ActionMailer::Base
  default from: Rails.configuration.Mail.IssueReporterEmailFrom
  default to: Rails.configuration.Mail.IssueReporterEmailTo

  def send_report(user, params)
    @user = user
    @params = params
    subject = 'Issue reported'
    subject += " by #{@user.email}" if @user
    mail(subject: subject)
  end
end
