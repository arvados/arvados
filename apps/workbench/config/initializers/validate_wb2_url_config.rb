# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'uri'

if Rails.configuration.workbench2_url
    begin
        if !URI.parse(Rails.configuration.workbench2_url).is_a?(URI::HTTP)
            Rails.logger.warn("workbench2_url config is not an HTTP URL: #{Rails.configuration.workbench2_url}")
            Rails.configuration.workbench2_url = false
        elsif /.*[\/]{2,}$/.match(Rails.configuration.workbench2_url)
            Rails.logger.warn("workbench2_url config shouldn't have multiple trailing slashes: #{Rails.configuration.workbench2_url}")
            Rails.configuration.workbench2_url = false
        end
    rescue URI::InvalidURIError
        Rails.logger.warn("workbench2_url config invalid URL: #{Rails.configuration.workbench2_url}")
        Rails.configuration.workbench2_url = false
    end
end
