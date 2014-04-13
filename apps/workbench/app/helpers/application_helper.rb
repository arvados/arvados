module ApplicationHelper
  def current_user
    controller.current_user
  end

  def self.match_uuid(uuid)
    /^([0-9a-z]{5})-([0-9a-z]{5})-([0-9a-z]{15})$/.match(uuid.to_s)
  end

  def current_api_host
    Rails.configuration.arvados_v1_base.gsub /https?:\/\/|\/arvados\/v1/,''
  end

  def render_content_from_database(markup)
    raw RedCloth.new(markup).to_html
  end

  def human_readable_bytes_html(n)
    return h(n) unless n.is_a? Fixnum

    orders = {
      1 => "bytes",
      1024 => "KiB",
      (1024*1024) => "MiB",
      (1024*1024*1024) => "GiB",
      (1024*1024*1024*1024) => "TiB"
    }

    orders.each do |k, v|
      sig = (n.to_f/k)
      if sig >=1 and sig < 1024
        if v == 'bytes'
          return "%i #{v}" % sig
        else
          return "%0.1f #{v}" % sig
        end
      end
    end
    
    return h(n)
      #raw = n.to_s
    #cooked = ''
    #while raw.length > 3
    #  cooked = ',' + raw[-3..-1] + cooked
    #  raw = raw[0..-4]
    #end
    #cooked = raw + cooked
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

        if opts[:friendly_name]
          if attrvalue.respond_to? :friendly_link_name
            link_name = attrvalue.friendly_link_name
          else
            begin
              link_name = resource_class.find(link_uuid).friendly_link_name
            rescue RuntimeError
              # If that lookup failed, the link will too. So don't make one.
              return attrvalue
            end
          end
        end
        if opts[:with_class_name]
          link_name = "#{resource_class.to_s}: #{link_name}"
        end
      end
      style_opts[:class] = (style_opts[:class] || '') + ' nowrap'
      link_to link_name, { controller: resource_class.to_s.tableize, action: 'show', id: link_uuid }, style_opts
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
      "data-url" => url_for(action: "update", id: object.uuid, controller: object.class.to_s.pluralize.underscore),
      "data-title" => "Update #{attr.gsub '_', ' '}",
      "data-name" => attr,
      "data-pk" => "{id: \"#{object.uuid}\", key: \"#{object.class.to_s.underscore}\"}",
      :class => "editable"
    }.merge(htmloptions)
  end

  def render_pipeline_component_attribute(object, attr, subattr, value_info, htmloptions={})
    datatype = nil
    required = true
    if value_info.is_a? Hash
      if value_info[:output_of]
        return raw("<span class='label label-default'>#{value_info[:output_of]}</span>")
      end
      if value_info[:dataclass]
        dataclass = value_info[:dataclass]
      end
      if value_info[:optional] != nil
        required = (value_info[:optional] != "true")
      end
      if value_info[:required] != nil
        required = value_info[:required]
      end
    end

    attrvalue = value_info
    if value_info.is_a? Hash
      if value_info[:value]
        attrvalue = value_info[:value]
      elsif value_info[:default]
        attrvalue = value_info[:default]
      else
        attrvalue = ''
      end
    end

    unless object.andand.attribute_editable? attr
      return link_to_if_arvados_object attrvalue
    end

    if dataclass
      begin
        dataclass = dataclass.constantize
      rescue NameError
      end
    else
      dataclass = ArvadosBase.resource_class_for_uuid(attrvalue)
    end

    if dataclass.andand.is_a?(Class)
      datatype = 'select'
    elsif dataclass == 'number'
      datatype = 'number'
    elsif attrvalue.is_a? Array
      datatype = 'text'
      attrvalue = attrvalue.to_s
    elsif attrvalue.is_a? Fixnum or attrvalue.is_a? Float
      datatype = 'number'
    elsif attrvalue.is_a? String
      datatype = 'text'
    end

    id = "#{object.uuid}-#{subattr.join('-')}"
    dn = "[#{attr}]"
    subattr.each do |a|
      dn += "[#{a}]"
    end
    if value_info.is_a? Hash
      dn += '[value]'
    end

    items = []
    if dataclass and dataclass.is_a? Class
      if attrvalue and !attrvalue.empty?
        items.append({name: attrvalue, uuid: attrvalue, type: dataclass.to_s})
      end
      #dataclass.where(uuid: attrvalue).each do |item|
      #  items.append({name: item.uuid, uuid: item.uuid, type: dataclass.to_s})
      #end
      dataclass.limit(10).each do |item|
        items.append({name: item.uuid, uuid: item.uuid, type: dataclass.to_s})
      end
    end

    lt = link_to attrvalue, '#', {
      "data-emptytext" => "none",
      "data-placement" => "bottom",
      "data-type" => datatype,
      "data-url" => url_for(action: "update", id: object.uuid, controller: object.class.to_s.pluralize.underscore, merge: true),
      "data-title" => "Set value for #{subattr[-1].to_s}",
      "data-name" => dn,
      "data-pk" => "{id: \"#{object.uuid}\", key: \"#{object.class.to_s.underscore}\"}",
      "data-showbuttons" => "false",
      "data-value" => attrvalue,
      :class => "editable #{'required' if required}",
      :id => id
    }.merge(htmloptions)

    lt += raw("\n<script>")
    
    if items.any?
      lt += raw("add_form_selection_sources(#{items.to_json});\n")
    end

    lt += raw("$('##{id}').editable({source: function() { return select_form_sources('#{dataclass}'); } });\n")

    lt += raw("</script>")

    lt 
  end
end
