# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'uri'

module ConfigValidators
  def self.validate_wb2_url_config
    if Rails.configuration.Services.Workbench2.ExternalURL != URI("")
      if !Rails.configuration.Services.Workbench2.ExternalURL.is_a?(URI::HTTP)
        raise "workbench2_url config is not an HTTP URL: #{Rails.configuration.Services.Workbench2.ExternalURL}"
      elsif /.*[\/]{2,}$/.match(Rails.configuration.Services.Workbench2.ExternalURL.to_s)
        raise "workbench2_url config shouldn't have multiple trailing slashes: #{Rails.configuration.Services.Workbench2.ExternalURL}"
      else
        return true
      end
    end
    return false
  end

  def self.validate_download_config
    if Rails.configuration.Services.WebDAV.ExternalURL == URI("") and Rails.configuration.Services.WebDAVDownload.ExternalURL == URI("")
      raise "Keep-web service must be configured in Services.WebDAV and/or Services.WebDAVDownload"
    end
  end
end
