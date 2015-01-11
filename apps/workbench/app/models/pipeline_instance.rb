class PipelineInstance < ArvadosBase
  attr_accessor :pipeline_template

  def self.goes_in_projects?
    true
  end

  def friendly_link_name lookup=nil
    pipeline_name = self.name
    if pipeline_name.nil? or pipeline_name.empty?
      template = if lookup and lookup[self.pipeline_template_uuid]
                   lookup[self.pipeline_template_uuid]
                 else
                   PipelineTemplate.where(uuid: self.pipeline_template_uuid).first
                 end
      if template
        template.name
      else
        self.uuid
      end
    else
      pipeline_name
    end
  end

  def content_summary
    begin
      PipelineTemplate.find(pipeline_template_uuid).name
    rescue
      super
    end
  end

  def update_job_parameters(new_params)
    self.components[:steps].each_with_index do |step, i|
      step[:params].each do |param|
        if new_params.has_key?(new_param_name = "#{i}/#{param[:name]}") or
            new_params.has_key?(new_param_name = "#{step[:name]}/#{param[:name]}") or
            new_params.has_key?(new_param_name = param[:name])
          param_type = :value
          %w(hash data_locator).collect(&:to_sym).each do |ptype|
            param_type = ptype if param.has_key? ptype
          end
          param[param_type] = new_params[new_param_name]
        end
      end
    end
  end

  def editable_attributes
    %w(name description components)
  end

  def attribute_editable?(name, ever=nil)
    if name.to_s == "components"
      (ever or %w(New Ready).include?(state)) and super
    else
      super
    end
  end

  def attributes_for_display
    super.reject { |k,v| k == 'components' }
  end

  def self.creatable?
    false
  end

  def component_input_title(component_name, input_name)
    component = components[component_name]
    return nil if component.nil?
    param_info = component[:script_parameters].andand[input_name.to_sym]
    if param_info.is_a?(Hash) and param_info[:title]
      param_info[:title]
    else
      "\"#{input_name.to_s}\" parameter for #{component[:script]} script in #{component_name} component"
    end
  end

  def textile_attributes
    [ 'description' ]
  end
end
