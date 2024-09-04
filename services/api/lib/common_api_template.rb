# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module CommonApiTemplate
  def self.included(base)
    base.acts_as_api
    base.class_eval do
      alias_method :as_api_response_orig, :as_api_response
      include InstanceMethods
    end
    base.extend(ClassMethods)
    base.api_accessible :common do |t|
      t.add :href
      t.add :kind
      t.add :etag
      t.add :uuid
      t.add :owner_uuid
      t.add :created_at
      t.add :modified_by_user_uuid
      t.add :modified_at
    end
  end

  module InstanceMethods
    # choose template based on opts[:for_user]
    def as_api_response(template=nil, opts={})
      if template.nil?
        user = opts[:for_user] || current_user
        if user.andand.is_admin and self.respond_to? :api_accessible_superuser
          template = :superuser
        else
          template = :user
        end
      end
      self.as_api_response_orig(template, opts)
    end
  end

  module ClassMethods
  end
end
