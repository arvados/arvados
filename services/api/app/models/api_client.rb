# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ApiClient < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  has_many :api_client_authorizations

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :url_prefix
    t.add :is_trusted
  end

  def is_trusted
    norm(self.url_prefix) == norm(Rails.configuration.Services.Workbench1.ExternalURL) ||
      norm(self.url_prefix) == norm(Rails.configuration.Services.Workbench2.ExternalURL) ||
      super
  end

  protected

  def norm url
    # normalize URL for comparison
    url = URI(url)
    if url.scheme == "https"
      url.port == "443"
    end
    if url.scheme == "http"
      url.port == "80"
    end
    url.path = "/"
    url
  end
end
