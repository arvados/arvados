# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'uri'

module ConfigValidators
    def self.validate_wb2_url_config
        if Rails.configuration.Services.Workbench2.ExternalURL != URI("")
          if !Rails.configuration.Services.Workbench2.ExternalURL.is_a?(URI::HTTP)
            Rails.logger.warn("workbench2_url config is not an HTTP URL: #{Rails.configuration.Services.Workbench2.ExternalURL}")
            Rails.configuration.Services.Workbench2.ExternalURL = URI("")
          elsif /.*[\/]{2,}$/.match(Rails.configuration.Services.Workbench2.ExternalURL.to_s)
            Rails.logger.warn("workbench2_url config shouldn't have multiple trailing slashes: #{Rails.configuration.Services.Workbench2.ExternalURL}")
            Rails.configuration.Services.Workbench2.ExternalURL = URI("")
          else
            return true
          end
        end
        return false
    end
end
