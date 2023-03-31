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
    (from_trusted_url && Rails.configuration.Login.IssueTrustedTokens) || super
  end

  protected

  def from_trusted_url
    norm_url_prefix = norm(self.url_prefix)

    [Rails.configuration.Services.Workbench1.ExternalURL,
     Rails.configuration.Services.Workbench2.ExternalURL,
     "https://controller.api.client.invalid"].each do |url|
      if norm_url_prefix == norm(url)
        return true
      end
    end

    Rails.configuration.Login.TrustedClients.keys.each do |url|
      trusted = norm(url)
      if norm_url_prefix == trusted
        return true
      end
      if trusted.host.to_s.starts_with?("*.") &&
         norm_url_prefix.to_s.starts_with?(trusted.scheme + "://") &&
         norm_url_prefix.to_s.ends_with?(trusted.to_s[trusted.scheme.length + 4...])
        return true
      end
    end

    false
  end

  def norm url
    # normalize URL for comparison
    url = URI(url.to_s)
    if url.scheme == "https" && url.port == ""
      url.port = "443"
    elsif url.scheme == "http" && url.port == ""
      url.port = "80"
    end
    url.path = "/"
    url.query = nil
    url.fragment = nil
    url
  end
end
