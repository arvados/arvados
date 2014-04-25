class AdminNotifier < ActionMailer::Base
  default from: Rails.configuration.admin_notifier_email_from

  def after_create(model, *args)
    self.generic_callback('after_create', model, *args)
  end

  def new_inactive_user(user)
    @user = user
    if not Rails.configuration.new_inactive_user_notification_recipients.empty? then
      mail(to: Rails.configuration.new_inactive_user_notification_recipients, subject: 'New inactive user notification')
    end
  end

  protected

  def generic_callback(callback_type, model, *args)
    model_specific_method = "#{callback_type}_#{model.class.to_s.underscore}".to_sym
    if self.respond_to?(model_specific_method,true)
      self.send model_specific_method, model, *args
    end
  end

  def all_admin_emails()
    User.
      where(is_admin: true).
      collect(&:email).
      compact.
      uniq.
      select { |e| e.match /\@/ }
  end

  def after_create_user(user, *args)
    @new_user = user
    @recipients = self.all_admin_emails
    logger.info "Sending mail to #{@recipients} about new user #{@new_user.uuid} (#{@new_user.full_name}, #{@new_user.email})"
    mail(template_name: __method__,
         to: @recipients,
         subject: "#{Rails.configuration.email_subject_prefix}New user: #{@new_user.full_name}, #{@new_user.email}"
        ).deliver
  end
end
