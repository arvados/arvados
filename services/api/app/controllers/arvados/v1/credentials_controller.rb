# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::CredentialsController < ApplicationController

  # "secret" is not returned in API calls, but we also want
  # to disallow its use in queries in general.

  def load_where_param
    super
    if @where[:secret]
      raise ArvadosModel::PermissionDeniedError.new "Cannot use 'secret' in where clause"
    end
  end

  def load_filters_param
    super
    @filters.map do |k|
      if k[0] =~ /secret/
        raise ArvadosModel::PermissionDeniedError.new "Cannot filter on 'secret'"
      end
    end
  end

  def load_limit_offset_order_params
    super
    @orders.each do |ord|
      if ord =~ /secret/
        raise ArvadosModel::PermissionDeniedError.new "Cannot order by 'secret'"
      end
    end
  end

  def self._secret_method_description
    "Fetch the secret part of the credential (can only be invoked by running containers)."
  end

  def secret
    c = Container.for_current_token
    if @object && c && c.state == "Running" && current_user.can?(read: @object)
      if Time.now >= @object.expires_at
        send_error("Credential has expired.", status: 403)
      else
        lg = Log.new(event_type: "secret_access")
        lg.object_uuid = @object.uuid
        lg.object_owner_uuid = @object.owner_uuid
        lg.properties = {
          "name": @object.name,
                         "credential_class": @object.credential_class,
                         "external_id": @object.external_id,
        }
        lg.save!
        send_json({"external_id" => @object.external_id, "secret" => @object.secret})
      end
    else
      send_error("Token is not associated with a container.", status: 403)
    end
  end
end
