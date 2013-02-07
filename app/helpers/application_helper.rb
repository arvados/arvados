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

  def link_to_if_orvos_object(attrvalue, attr, object)
    if (resource_class = OrvosBase::resource_class_for_uuid(attrvalue, attr, object))
      link_to "#{resource_class.to_s} #{attrvalue}", { controller: resource_class.to_s.camelize(:lower).pluralize, action: 'show', id: attrvalue }
    else
      attrvalue
    end
  end
end
