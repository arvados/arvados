module ApplicationHelper
  def current_user
    controller.current_user
  end

  def current_api_host
    Rails.configuration.arvados_v1_base.gsub /https?:\/\/|\/arvados\/v1/,''
  end

  def render_content_from_database(markup)
    raw RedCloth.new(markup).to_html
  end

  def human_readable_bytes_html(n)
    return h(n) unless n.is_a? Fixnum
    raw = n.to_s
    cooked = ''
    while raw.length > 3
      cooked = ',' + raw[-3..-1] + cooked
      raw = raw[0..-4]
    end
    cooked = raw + cooked
  end

  def resource_class_for_uuid(attrvalue, opts={})
    ArvadosBase::resource_class_for_uuid(attrvalue, opts)
  end

  def link_to_if_arvados_object(attrvalue, opts={}, style_opts={})
    if (resource_class = resource_class_for_uuid(attrvalue, opts))
      link_uuid = attrvalue.is_a?(ArvadosBase) ? attrvalue.uuid : attrvalue
      link_name = opts[:link_text]
      if !link_name
        link_name = link_uuid

        if opts[:friendly_name] and resource_class.column_names.include? "name" and resource_class.find(link_uuid).name != nil and not resource_class.find(link_uuid).name.empty?
          link_name = "#{resource_class.to_s} #{resource_class.find(link_uuid).name}"
        elsif opts[:friendly_name] and resource_class.column_names.include? "hostname" and resource_class.find(link_uuid).hostname != nil and not resource_class.find(link_uuid).hostname.empty?
          link_name = "#{resource_class.to_s} #{resource_class.find(link_uuid).hostname}"
        elsif opts[:friendly_name] and resource_class.column_names.include? "first_name"
          link_name = "#{resource_class.to_s} #{resource_class.find(link_uuid).first_name} #{resource_class.find(link_uuid).last_name}"
        else
          if opts[:with_class_name]
            link_name = "#{resource_class.to_s} #{link_name}"
          end
        end
      end
      link_to link_name, { controller: resource_class.to_s.underscore.pluralize, action: 'show', id: link_uuid }, style_opts
    else
      attrvalue
    end
  end

  def render_editable_attribute(object, attr, attrvalue=nil, htmloptions={})
    attrvalue = object.send(attr) if attrvalue.nil?
    return attrvalue if !object.attribute_editable? attr

    input_type = 'text'
    case object.class.attribute_info[attr.to_sym].andand[:type]
    when 'text'
      input_type = 'textarea'
    when 'datetime'
      input_type = 'date'
    else
      input_type = 'text'
    end

    attrvalue = attrvalue.to_json if attrvalue.is_a? Hash or attrvalue.is_a? Array

    link_to attrvalue.to_s, '#', {
      "data-emptytext" => "none",
      "data-placement" => "bottom",
      "data-type" => input_type,
      "data-resource" => object.class.to_s.underscore,
      "data-name" => attr,
      "data-url" => url_for(action: "update", id: object.uuid, controller: object.class.to_s.pluralize.underscore),
      "data-original-title" => "Update #{attr.gsub '_', ' '}",
      :class => "editable"
    }.merge(htmloptions)
  end
end
