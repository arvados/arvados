# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module PipelineComponentsHelper
  def render_pipeline_components(template_suffix, fallback=nil, locals={})
    begin
      render(partial: "pipeline_instances/show_components_#{template_suffix}",
             locals: locals)
    rescue => e
      logger.error "#{e.inspect}"
      logger.error "#{e.backtrace.join("\n\t")}"
      case fallback
      when :json
        render(partial: "pipeline_instances/show_components_json",
               locals: {error_name: e.inspect, backtrace: e.backtrace.join("\n\t")})
      end
    end
  end
end
