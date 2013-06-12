class UsersController < ApplicationController
  skip_before_filter :find_object_by_uuid, :only => :welcome

  def welcome
    if current_user
      redirect_to home_user_path(current_user.uuid)
    else
      redirect_to $arvados_api_client.arvados_login_url(return_to: request.url)
    end
  end

  def home
    @my_ssh_keys = AuthorizedKey.where(authorized_user: current_user.uuid)
    @my_vm_perms = Link.where(tail_uuid: current_user.uuid, head_kind: 'arvados#virtual_machine', link_class: 'permission', name: 'can_login')
    @my_repo_perms = Link.where(tail_uuid: current_user.uuid, head_kind: 'arvados#repository', link_class: 'permission', name: 'can_write')
  end
end
