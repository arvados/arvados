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

  def render_markup(markup)
    raw RedCloth.new(markup.to_s).to_html(:refs_arvados, :textile) if markup
  end

  def human_readable_bytes_html(n)
    return h(n) unless n.is_a? Fixnum
    return "0 bytes" if (n == 0)

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

  ##
  # Returns HTML that links to the Arvados object specified in +attrvalue+
  # Provides various output control and styling options.
  #
  # +attrvalue+ an Arvados model object or uuid
  #
  # +opts+ a set of flags to control output:
  #
  # [:link_text] the link text to use (may include HTML), overrides everything else
  #
  # [:friendly_name] whether to use the "friendly" name in the link text (by
  # calling #friendly_link_name on the object), otherwise use the uuid
  #
  # [:with_class_name] prefix the link text with the class name of the model
  #
  # [:no_tags] disable tags in the link text (default is to show tags).
  # Currently tags are only shown for Collections.
  #
  # [:thumbnail] if the object is a collection, show an image thumbnail if the
  # collection consists of a single image file.
  #
  # [:no_link] don't create a link, just return the link text
  #
  # +style_opts+ additional HTML properties for the anchor tag, passed to link_to
  #
  def link_to_if_arvados_object(attrvalue, opts={}, style_opts={})
    if (resource_class = resource_class_for_uuid(attrvalue, opts))
      if attrvalue.is_a? ArvadosBase
        object = attrvalue
        link_uuid = attrvalue.uuid
      else
        object = nil
        link_uuid = attrvalue
      end
      link_name = opts[:link_text]
      tags = ""
      if !link_name
        link_name = object.andand.default_name || resource_class.default_name

        if opts[:friendly_name]
          if attrvalue.respond_to? :friendly_link_name
            link_name = attrvalue.friendly_link_name opts[:lookup]
          else
            begin
              if resource_class.name == 'Collection'
                link_name = collections_for_object(link_uuid).andand.first.andand.friendly_link_name
              else
                link_name = object_for_dataclass(resource_class, link_uuid).andand.friendly_link_name
              end
            rescue ArvadosApiClient::NotFoundException
              # If that lookup failed, the link will too. So don't make one.
              return attrvalue
            end
          end
        end
        if link_name.nil? or link_name.empty?
          link_name = attrvalue
        end
        if opts[:with_class_name]
          link_name = "#{resource_class.to_s}: #{link_name}"
        end
        if !opts[:no_tags] and resource_class == Collection
          links_for_object(link_uuid).each do |tag|
            if tag.link_class.in? ["tag", "identifier"]
              tags += ' <span class="label label-info">'
              tags += link_to tag.name, controller: "links", filters: [["link_class", "=", "tag"], ["name", "=", tag.name]].to_json
              tags += '</span>'
            end
          end
        end
        if opts[:thumbnail] and resource_class == Collection
          # add an image thumbnail if the collection consists of a single image file.
          collections_for_object(link_uuid).each do |c|
            if c.files.length == 1 and CollectionsHelper::is_image c.files.first[1]
              link_name += " "
              link_name += image_tag "#{url_for c}/#{CollectionsHelper::file_path c.files.first}", style: "height: 4em; width: auto"
            end
          end
        end
      end
      style_opts[:class] = (style_opts[:class] || '') + ' nowrap'
      if opts[:no_link]
        raw(link_name)
      else
        (link_to raw(link_name), { controller: resource_class.to_s.tableize, action: 'show', id: ((opts[:name_link].andand.uuid) || link_uuid) }, style_opts) + raw(tags)
      end
    else
      # just return attrvalue if it is not recognizable as an Arvados object or uuid.
      if attrvalue.nil? or (attrvalue.is_a? String and attrvalue.empty?)
        "(none)"
      else
        attrvalue
      end
    end
  end

  def render_editable_attribute(object, attr, attrvalue=nil, htmloptions={})
    attrvalue = object.send(attr) if attrvalue.nil?
    if not object.attribute_editable?(attr)
      if attrvalue && attrvalue.length > 0
        return render_attribute_as_textile( object, attr, attrvalue, false )
      else
        return (attr == 'name' and object.andand.default_name) ||
                '(none)'
      end
    end

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
    rendervalue = render_attribute_as_textile( object, attr, attrvalue, false )

    ajax_options = {
      "data-pk" => {
        id: object.uuid,
        key: object.class.to_s.underscore
      }
    }
    if object.uuid
      ajax_options['data-url'] = url_for(action: "update", id: object.uuid, controller: object.class.to_s.pluralize.underscore)
    else
      ajax_options['data-url'] = url_for(action: "create", controller: object.class.to_s.pluralize.underscore)
      ajax_options['data-pk'][:defaults] = object.attributes
    end
    ajax_options['data-pk'] = ajax_options['data-pk'].to_json
    @unique_id ||= (Time.now.to_f*1000000).to_i
    span_id = object.uuid.to_s + '-' + attr.to_s + '-' + (@unique_id += 1).to_s

    span_tag = content_tag 'span', rendervalue, {
      "data-emptytext" => '(none)',
      "data-placement" => "bottom",
      "data-type" => input_type,
      "data-title" => "Edit #{attr.to_s.gsub '_', ' '}",
      "data-name" => attr,
      "data-object-uuid" => object.uuid,
      "data-toggle" => "manual",
      "data-value" => attrvalue,
      "id" => span_id,
      :class => "editable #{is_textile?( object, attr ) ? 'editable-textile' : ''}"
    }.merge(htmloptions).merge(ajax_options)
    edit_button = raw('<a href="#" class="btn btn-xs btn-default btn-nodecorate" data-toggle="x-editable tooltip" data-toggle-selector="#' + span_id + '" data-placement="top" title="' + (htmloptions[:tiptitle] || 'edit') + '"><i class="fa fa-fw fa-pencil"></i></a>')
    if htmloptions[:btnplacement] == :left
      edit_button + ' ' + span_tag
    else
      span_tag + ' ' + edit_button
    end
  end

  def render_pipeline_component_attribute(object, attr, subattr, value_info, htmloptions={})
    datatype = nil
    required = true
    attrvalue = value_info

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

      # Pick a suitable attrvalue to show as the current value (i.e.,
      # the one that would be used if we ran the pipeline right now).
      if value_info[:value]
        attrvalue = value_info[:value]
      elsif value_info[:default]
        attrvalue = value_info[:default]
      else
        attrvalue = ''
      end
      preconfigured_search_str = value_info[:search_for]
    end

    if not object.andand.attribute_editable?(attr)
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

    id = "#{object.uuid}-#{subattr.join('-')}"
    dn = "[#{attr}]"
    subattr.each do |a|
      dn += "[#{a}]"
    end
    if value_info.is_a? Hash
      dn += '[value]'
    end

    if (dataclass == Collection) or (dataclass == File)
      selection_param = object.class.to_s.underscore + dn
      display_value = attrvalue
      if value_info.is_a?(Hash)
        if (link = Link.find? value_info[:link_uuid])
          display_value = link.name
        elsif value_info[:link_name]
          display_value = value_info[:link_name]
        elsif value_info[:selection_name]
          display_value = value_info[:selection_name]
        end
      end
      if (attr == :components) and (subattr.size > 2)
        chooser_title = "Choose a #{dataclass == Collection ? 'dataset' : 'file'} for #{object.component_input_title(subattr[0], subattr[2])}:"
      else
        chooser_title = "Choose a #{dataclass == Collection ? 'dataset' : 'file'}:"
      end
      modal_path = choose_collections_path \
      ({ title: chooser_title,
         filters: [['owner_uuid', '=', object.owner_uuid]].to_json,
         action_name: 'OK',
         action_href: pipeline_instance_path(id: object.uuid),
         action_method: 'patch',
         preconfigured_search_str: (preconfigured_search_str || ""),
         action_data: {
           merge: true,
           use_checkbox_selection: dataclass == File ? true : false,
           selection_param: selection_param,
           success: 'page-refresh'
         }.to_json,
        })
      return content_tag('div', :class => 'input-group') do
        html = text_field_tag(dn, display_value,
                              :class =>
                              "form-control #{'required' if required}")
        html + content_tag('span', :class => 'input-group-btn') do
          link_to('Choose',
                  modal_path,
                  { :class => "btn btn-primary",
                    :remote => true,
                    :method => 'get',
                  })
        end
      end
    end

    if dataclass.andand.is_a?(Class)
      datatype = 'select'
    elsif dataclass == 'number'
      datatype = 'number'
    elsif attrvalue.is_a? Array
      # TODO: find a way to edit arrays with x-editable
      return attrvalue
    elsif attrvalue.is_a? Fixnum or attrvalue.is_a? Float
      datatype = 'number'
    elsif attrvalue.is_a? String
      datatype = 'text'
    end

    # preload data
    preload_uuids = []
    items = []
    selectables = []

    attrtext = attrvalue
    if dataclass.is_a? Class and dataclass < ArvadosBase
      objects = get_n_objects_of_class dataclass, 10
      objects.each do |item|
        items << item
        preload_uuids << item.uuid
      end
      if attrvalue and !attrvalue.empty?
        preload_uuids << attrvalue
      end
      preload_links_for_objects preload_uuids

      if attrvalue and !attrvalue.empty?
        links_for_object(attrvalue).each do |link|
          if link.link_class.in? ["tag", "identifier"]
            attrtext += " [#{link.name}]"
          end
        end
        selectables.append({name: attrtext, uuid: attrvalue, type: dataclass.to_s})
      end
      itemuuids = []
      items.each do |item|
        itemuuids << item.uuid
        selectables.append({name: item.uuid, uuid: item.uuid, type: dataclass.to_s})
      end

      itemuuids.each do |itemuuid|
        links_for_object(itemuuid).each do |link|
          if link.link_class.in? ["tag", "identifier"]
            selectables.each do |selectable|
              if selectable['uuid'] == link.head_uuid
                selectable['name'] += ' [' + link.name + ']'
              end
            end
          end
        end
      end
    end

    lt = link_to attrtext, '#', {
      "data-emptytext" => "none",
      "data-placement" => "bottom",
      "data-type" => datatype,
      "data-url" => url_for(action: "update", id: object.uuid, controller: object.class.to_s.pluralize.underscore, merge: true),
      "data-title" => "Set value for #{subattr[-1].to_s}",
      "data-name" => dn,
      "data-pk" => "{id: \"#{object.uuid}\", key: \"#{object.class.to_s.underscore}\"}",
      "data-value" => attrvalue,
      # "clear" button interferes with form-control's up/down arrows
      "data-clear" => false,
      :class => "editable #{'required' if required} form-control",
      :id => id
    }.merge(htmloptions)

    lt += raw("\n<script>")

    if selectables.any?
      lt += raw("add_form_selection_sources(#{selectables.to_json});\n")
    end

    lt += raw("$('[data-name=\"#{dn}\"]').editable({source: function() { return select_form_sources('#{dataclass}'); } });\n")

    lt += raw("</script>")

    lt
  end

  def render_arvados_object_list_start(list, button_text, button_href,
                                       params={}, *rest, &block)
    show_max = params.delete(:show_max) || 3
    params[:class] ||= 'btn btn-xs btn-default'
    list[0...show_max].each { |item| yield item }
    unless list[show_max].nil?
      link_to(h(button_text) +
              raw(' &nbsp; <i class="fa fa-fw fa-arrow-circle-right"></i>'),
              button_href, params, *rest)
    end
  end

  def render_controller_partial partial, opts
    cname = opts.delete :controller_name
    begin
      render opts.merge(partial: "#{cname}/#{partial}")
    rescue ActionView::MissingTemplate
      render opts.merge(partial: "application/#{partial}")
    end
  end

  RESOURCE_CLASS_ICONS = {
    "Collection" => "fa-archive",
    "Group" => "fa-users",
    "Human" => "fa-male",  # FIXME: Use a more inclusive icon.
    "Job" => "fa-gears",
    "KeepDisk" => "fa-hdd-o",
    "KeepService" => "fa-exchange",
    "Link" => "fa-arrows-h",
    "Node" => "fa-cloud",
    "PipelineInstance" => "fa-gears",
    "PipelineTemplate" => "fa-gears",
    "Repository" => "fa-code-fork",
    "Specimen" => "fa-flask",
    "Trait" => "fa-clipboard",
    "User" => "fa-user",
    "VirtualMachine" => "fa-terminal",
  }
  DEFAULT_ICON_CLASS = "fa-cube"

  def fa_icon_class_for_class(resource_class, default=DEFAULT_ICON_CLASS)
    RESOURCE_CLASS_ICONS.fetch(resource_class.to_s, default)
  end

  def fa_icon_class_for_uuid(uuid, default=DEFAULT_ICON_CLASS)
    fa_icon_class_for_class(resource_class_for_uuid(uuid), default)
  end

  def fa_icon_class_for_object(object, default=DEFAULT_ICON_CLASS)
    case class_name = object.class.to_s
    when "Group"
      object.group_class ? 'fa-folder' : 'fa-users'
    else
      RESOURCE_CLASS_ICONS.fetch(class_name, default)
    end
  end

  def chooser_preview_url_for object, use_checkbox_selection=false
    case object.class.to_s
    when 'Collection'
      polymorphic_path(object, tab_pane: 'chooser_preview', use_checkbox_selection: use_checkbox_selection)
    else
      nil
    end
  end

  def render_attribute_as_textile( object, attr, attrvalue, truncate )
    if attrvalue && (is_textile? object, attr)
      markup = render_markup attrvalue
      markup = markup[0,markup.index('</p>')+4] if (truncate && markup.index('</p>'))
      return markup
    else
      return attrvalue
    end
  end

  def render_localized_date(date, opts="")
    raw("<span class='utc-date' data-utc-date='#{date}' data-utc-date-opts='noseconds'>#{date}</span>")
  end

private
  def is_textile?( object, attr )
    is_textile = object.textile_attributes.andand.include?(attr)
  end
end
