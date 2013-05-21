module ApplicationHelper
  def current_user
    controller.current_user
  end

  def act_as_system_user
    if not $system_user
      Thread.current[:user] = User.new(is_admin: true)
      sysuser_id = [Server::Application.config.uuid_prefix,
                    User.uuid_prefix,
                    '000000000000000'].join('-')
      $system_user = User.where('uuid=?', sysuser_id).first
      if !$system_user
        $system_user = User.new(uuid: sysuser_id,
                                is_admin: true,
                                email: 'root',
                                first_name: 'root',
                                last_name: '')
        $system_user.save!
        $system_user.reload
      end
    end
    Thread.current[:user] = $system_user
  end
end
