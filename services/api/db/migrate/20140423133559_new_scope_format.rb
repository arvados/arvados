# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# At the time we introduced scopes everywhere, VirtualMachinesController
# recognized scopes that gave the URL for a VM to grant access to that VM's
# login list.  This migration converts those VM-specific scopes to the new
# general format, and back.

class NewScopeFormat < ActiveRecord::Migration
  include CurrentApiClient

  VM_PATH_REGEX =
    %r{(/arvados/v1/virtual_machines/[0-9a-z]{5}-[0-9a-z]{5}-[0-9a-z]{15})}
  OLD_SCOPE_REGEX = %r{^https?://[^/]+#{VM_PATH_REGEX.source}$}
  NEW_SCOPE_REGEX = %r{^GET #{VM_PATH_REGEX.source}/logins$}

  def fix_scopes_matching(regex)
    act_as_system_user
    ApiClientAuthorization.find_each do |auth|
      auth.scopes = auth.scopes.map do |scope|
        if match = regex.match(scope)
          yield match
        else
          scope
        end
      end
      auth.save!
    end
  end

  def up
    fix_scopes_matching(OLD_SCOPE_REGEX) do |match|
      "GET #{match[1]}/logins"
    end
  end

  def down
    case Rails.env
    when 'test'
      hostname = 'www.example.com'
    else
      require 'socket'
      hostname = Socket.gethostname
    end
    fix_scopes_matching(NEW_SCOPE_REGEX) do |match|
      Rails.application.routes.url_for(controller: 'virtual_machines',
                                       uuid: match[1].split('/').last,
                                       host: hostname, protocol: 'https')
    end
  end
end
