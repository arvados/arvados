class VirtualMachine < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate

  api_accessible :superuser, :extend => :common do |t|
    t.add :hostname
  end

  protected

  def permission_to_create
    current_user and current_user.is_admin
  end
  def permission_to_update
    current_user and current_user.is_admin
  end
end
