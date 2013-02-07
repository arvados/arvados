module ApplicationHelper
  def current_user
    controller.current_user
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

  def link_to_if_orvos_object(attrvalue, opts={})
    if (resource_class = OrvosBase::resource_class_for_uuid(attrvalue, opts[:referring_attr], opts[:referring_object]))
      link_uuid = attrvalue.is_a?(OrvosBase) ? attrvalue.uuid : attrvalue
      link_name = link_uuid
      if !opts[:with_prefixes]
        link_name = link_name.sub /^.{5}-.{5}-/, ''
      end
      if opts[:with_class_name]
        link_name = "#{resource_class.to_s} #{link_name}"
      end
      link_to link_name, { controller: resource_class.to_s.underscore.pluralize, action: 'show', id: link_uuid }
    else
      attrvalue
    end
  end
end
