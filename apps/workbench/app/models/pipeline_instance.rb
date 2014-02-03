class PipelineInstance < ArvadosBase
  attr_accessor :pipeline_template

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

  def attribute_editable?(attr)
    attr == 'name'
  end

  def attributes_for_display
    super.reject { |k,v| k == 'components' }
  end
end
