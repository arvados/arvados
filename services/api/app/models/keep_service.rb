class KeepService < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  api_accessible :user, extend: :common do |t|
    t.add  :service_host
    t.add  :service_port
    t.add  :service_ssl_flag
    t.add  :service_type
    t.add  :read_only
  end
  api_accessible :superuser, :extend => :user do |t|
  end

  protected

  def permission_to_create
    current_user.andand.is_admin
  end

  def permission_to_update
    current_user.andand.is_admin
  end
end
