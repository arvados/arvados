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
    (from_trusted_url && Rails.configuration.Login.TokenLifetime == 0) || super
  end

  protected

  def from_trusted_url
    norm_url_prefix = norm(self.url_prefix)
    norm_url_prefix == norm(Rails.configuration.Services.Workbench1.ExternalURL) or
      norm_url_prefix == norm(Rails.configuration.Services.Workbench2.ExternalURL) or
      norm_url_prefix == norm("https://controller.api.client.invalid")
  end

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
