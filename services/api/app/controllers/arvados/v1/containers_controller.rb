class Arvados::V1::ContainersController < ApplicationController
  accept_attribute_as_json :environment, Hash
  accept_attribute_as_json :mounts, Hash
  accept_attribute_as_json :runtime_constraints, Hash
  accept_attribute_as_json :command, Array

  def auth
    if @object.locked_by_uuid != Thread.current[:api_client_authorization].uuid
      raise ArvadosModel::PermissionDeniedError.new("Not locked by your token")
    end
    @object = @object.auth
    show
  end

  # Updates use row locking to resolve races between multiple
  # dispatchers trying to lock the same container.
  def update
    @object.with_lock do
      super
    end
  end
end
