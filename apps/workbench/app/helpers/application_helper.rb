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

  def link_to_if_arvados_object(attrvalue, opts={}, style_opts={})
    if (resource_class = ArvadosBase::resource_class_for_uuid(attrvalue, opts))
      link_uuid = attrvalue.is_a?(ArvadosBase) ? attrvalue.uuid : attrvalue
      link_name = opts[:link_text]
      if !link_name
        link_name = link_uuid
        if !opts[:with_prefixes]
          link_name = link_name.sub /^.{5}-.{5}-/, ''
        end
        if opts[:with_class_name]
          link_name = "#{resource_class.to_s} #{link_name}"
        end
        style_opts = style_opts.merge(style: 'font-family: monospace')
      end
      link_to link_name, { controller: resource_class.to_s.underscore.pluralize, action: 'show', id: link_uuid }, style_opts
    else
      attrvalue
    end
  end
end
