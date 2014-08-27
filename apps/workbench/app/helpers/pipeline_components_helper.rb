module PipelineComponentsHelper
  def render_pipeline_components(template_suffix, fallback=nil, locals={})
    begin
      render(partial: "pipeline_instances/show_components_#{template_suffix}",
             locals: locals)
    rescue Exception => e
      logger.error e.inspect
      case fallback
      when :json
        render(partial: "pipeline_instances/show_components_json")
      end
    end
  end
end
