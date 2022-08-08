# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module ApplicationHelper
  def current_user
    controller.current_user
  end

  def self.match_uuid(uuid)
    /^([0-9a-z]{5})-([0-9a-z]{5})-([0-9a-z]{15})$/.match(uuid.to_s)
  end

  def current_api_host
    if Rails.configuration.Services.Controller.ExternalURL.port == 443
      "#{Rails.configuration.Services.Controller.ExternalURL.hostname}"
    else
      "#{Rails.configuration.Services.Controller.ExternalURL.hostname}:#{Rails.configuration.Services.Controller.ExternalURL.port}"
    end
  end

  def current_uuid_prefix
    Rails.configuration.ClusterID
  end

  def render_markup(markup)
    allowed_tags = Rails::Html::Sanitizer.white_list_sanitizer.allowed_tags + %w(table tbody th tr td col colgroup caption thead tfoot)
    sanitize(raw(RedCloth.new(markup.to_s).to_html(:refs_arvados, :textile)), tags: allowed_tags) if markup
  end

  def human_readable_bytes_html(n)
    return h(n) unless n.is_a? Integer
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
  end

  def resource_class_for_uuid(attrvalue, opts={})
    ArvadosBase::resource_class_for_uuid(attrvalue, opts)
  end

  # When using {remote:true}, or using {method:...} to use an HTTP
  # method other than GET, move the target URI from href to
  # data-remote-href. Otherwise, browsers offer features like "open in
  # new window" and "copy link address" which bypass Rails' click
  # handler and therefore end up at incorrect/nonexistent routes (by
  # ignoring data-method) and expect to receive pages rather than
  # javascript responses.
  #
  # See assets/javascripts/link_to_remote.js for supporting code.
  def link_to *args, &block
    if (args.last and args.last.is_a? Hash and
        (args.last[:remote] or
         (args.last[:method] and
          args.last[:method].to_s.upcase != 'GET')))
      if Rails.env.test?
        # Capybara/phantomjs can't click_link without an href, even if
        # the click handler means it never gets used.
        raw super.gsub(' href="', ' href="#" data-remote-href="')
      else
        # Regular browsers work as desired: users can click A elements
        # without hrefs, and click handlers fire; but there's no "copy
        # link address" option in the right-click menu.
        raw super.gsub(' href="', ' data-remote-href="')
      end
    else
      super
    end
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
                if CollectionsHelper.match(link_uuid)
                  link_name = collection_for_pdh(link_uuid).andand.first.andand.portable_data_hash
                else
                  link_name = collections_for_object(link_uuid).andand.first.andand.friendly_link_name
                end
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
      if opts[:no_link] or (resource_class == User && !current_user)
        raw(link_name)
      else
        controller_class = resource_class.to_s.tableize
        if controller_class.eql?('groups') and (object.andand.group_class.eql?('project') or object.andand.group_class.eql?('filter'))
          controller_class = 'projects'
        end
        (link_to raw(link_name), { controller: controller_class, action: 'show', id: ((opts[:name_link].andand.uuid) || link_uuid) }, style_opts) + raw(tags)
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

  def link_to_arvados_object_if_readable(attrvalue, link_text_if_not_readable, opts={})
    resource_class = resource_class_for_uuid(attrvalue.split('/')[0]) if attrvalue.is_a?(String)
    if !resource_class
      return link_to_if_arvados_object attrvalue, opts
    end

    readable = object_readable attrvalue, resource_class
    if readable
      link_to_if_arvados_object attrvalue, opts
    elsif opts[:required] and current_user # no need to show this for anonymous user
      raw('<div><input type="text" style="border:none;width:100%;background:#ffdddd" disabled=true class="required unreadable-input" value="') + link_text_if_not_readable + raw('" ></input></div>')
    else
      link_text_if_not_readable
    end
  end

  # This method takes advantage of preloaded collections and objects.
  # Hence you can improve performance by first preloading objects
  # related to the page context before using this method.
  def object_readable attrvalue, resource_class=nil
    # if it is a collection filename, check readable for the locator
    attrvalue = attrvalue.split('/')[0] if attrvalue

    resource_class = resource_class_for_uuid(attrvalue) if resource_class.nil?
    return if resource_class.nil?

    return_value = nil
    if resource_class.to_s == 'Collection'
      if CollectionsHelper.match(attrvalue)
        found = collection_for_pdh(attrvalue)
        return_value = found.first if found.any?
      else
        found = collections_for_object(attrvalue)
        return_value = found.first if found.any?
      end
    else
      return_value = object_for_dataclass(resource_class, attrvalue)
    end
    return_value
  end

  # Render an editable attribute with the attrvalue of the attr.
  # The htmloptions are added to the editable element's list of attributes.
  # The nonhtml_options are only used to customize the display of the element.
  def render_editable_attribute(object, attr, attrvalue=nil, htmloptions={}, nonhtml_options={})
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
    opt_selection = nil
    attrtype = object.class.attribute_info[attr.to_sym].andand[:type]
    if attrtype == 'text' or attr == 'description'
      input_type = 'textarea'
    elsif attrtype == 'datetime'
      input_type = 'date'
    elsif attrtype == 'boolean'
      input_type = 'select'
      opt_selection = ([{value: "true", text: "true"}, {value: "false", text: "false"}]).to_json
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
      "data-source" => opt_selection,
      "data-title" => "Edit #{attr.to_s.gsub '_', ' '}",
      "data-name" => htmloptions['selection_name'] || attr,
      "data-object-uuid" => object.uuid,
      "data-toggle" => "manual",
      "data-value" => htmloptions['data-value'] || attrvalue,
      "id" => span_id,
      :class => "editable #{is_textile?( object, attr ) ? 'editable-textile' : ''}"
    }.merge(htmloptions).merge(ajax_options)

    edit_tiptitle = 'edit'
    edit_tiptitle = 'Warning: do not use hyphens in the repository name as they will be stripped' if (object.class.to_s == 'Repository' and attr == 'name')

    edit_button = raw('<a href="#" class="btn btn-xs btn-' + (nonhtml_options[:btnclass] || 'default') + ' btn-nodecorate" data-toggle="x-editable tooltip" data-toggle-selector="#' + span_id + '" data-placement="top" title="' + (nonhtml_options[:tiptitle] || edit_tiptitle) + '"><i class="fa fa-fw fa-pencil"></i>' + (nonhtml_options[:btntext] || '') + '</a>')

    if nonhtml_options[:btnplacement] == :left
      edit_button + ' ' + span_tag
    elsif nonhtml_options[:btnplacement] == :top
      edit_button + raw('<br/>') + span_tag
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
      return link_to_arvados_object_if_readable(attrvalue, attrvalue, {friendly_name: true, required: required})
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
        elsif (sn = value_info[:selection_name]) && sn != ""
          display_value = sn
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
           use_preview_selection: dataclass == File ? true : nil,
           selection_param: selection_param,
           success: 'page-refresh'
         }.to_json,
        })

      return content_tag('div', :class => 'input-group') do
        html = text_field_tag(dn, display_value,
                              :class =>
                              "form-control #{'required' if required} #{'unreadable-input' if attrvalue.present? and !object_readable(attrvalue, Collection)}")
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

    if attrvalue.is_a? String
      datatype = 'text'
    elsif attrvalue.is_a?(Array) or dataclass.andand.is_a?(Class)
      # TODO: find a way to edit with x-editable
      return attrvalue
    end

    # When datatype is a String or Fixnum, link_to the attrvalue
    lt = link_to attrvalue, '#', {
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

    lt
  end

  def get_cwl_main(workflow)
    if workflow[:"$graph"].nil?
      return workflow
    else
      workflow[:"$graph"].each do |tool|
        if tool[:id] == "#main"
          return tool
        end
      end
    end
  end

  def get_cwl_inputs(workflow)
    get_cwl_main(workflow)[:inputs]
  end


  def cwl_shortname(id)
    if id[0] == "#"
      id = id[1..-1]
    end
    return id.split("/")[-1]
  end

  def cwl_input_info(input_schema)
    required = !(input_schema[:type].include? "null")
    if input_schema[:type].is_a? Array
      primary_type = input_schema[:type].select { |n| n != "null" }[0]
    elsif input_schema[:type].is_a? String
      primary_type = input_schema[:type]
    elsif input_schema[:type].is_a? Hash
      primary_type = input_schema[:type]
    end
    param_id = cwl_shortname(input_schema[:id])
    return required, primary_type, param_id
  end

  def cwl_input_value(object, input_schema, set_attr_path)
    dn = ""
    attrvalue = object
    set_attr_path.each do |a|
      dn += "[#{a}]"
      attrvalue = attrvalue[a.to_sym]
    end
    return dn, attrvalue
  end

  def cwl_inputs_required(object, inputs_schema, set_attr_path)
    r = 0
    inputs_schema.each do |input|
      required, _, param_id = cwl_input_info(input)
      _, attrvalue = cwl_input_value(object, input, set_attr_path + [param_id])
      r += 1 if required and attrvalue.nil?
    end
    r
  end

  def render_cwl_input(object, input_schema, set_attr_path, htmloptions={})
    required, primary_type, param_id = cwl_input_info(input_schema)

    dn, attrvalue = cwl_input_value(object, input_schema, set_attr_path + [param_id])
    attrvalue = if attrvalue.nil? then "" else attrvalue end

    id = "#{object.uuid}-#{param_id}"

    opt_empty_selection = if required then [] else [{value: "", text: ""}] end

    if ["Directory", "File"].include? primary_type
      chooser_title = "Choose a #{primary_type == 'Directory' ? 'dataset' : 'file'}:"
      selection_param = object.class.to_s.underscore + dn
      if attrvalue.is_a? Hash
        display_value = attrvalue[:"http://arvados.org/cwl#collectionUUID"] || attrvalue[:"arv:collection"] || attrvalue[:location]
        re = CollectionsHelper.match_uuid_with_optional_filepath(display_value)
        locationre = CollectionsHelper.match(attrvalue[:location][5..-1])
        if re
          if locationre and locationre[4]
            display_value = "#{Collection.find(re[1]).name} / #{locationre[4][1..-1]}"
          else
            display_value = Collection.find(re[1]).name
          end
        end
      end
      modal_path = choose_collections_path \
      ({ title: chooser_title,
         filters: [['owner_uuid', '=', object.owner_uuid]].to_json,
         action_name: 'OK',
         action_href: container_request_path(id: object.uuid),
         action_method: 'patch',
         preconfigured_search_str: "",
         action_data: {
           merge: true,
           use_preview_selection: primary_type == 'File' ? true : nil,
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
    elsif "boolean" == primary_type
      return link_to attrvalue.to_s, '#', {
                     "data-emptytext" => "none",
                     "data-placement" => "bottom",
                     "data-type" => "select",
                     "data-source" => (opt_empty_selection + [{value: "true", text: "true"}, {value: "false", text: "false"}]).to_json,
                     "data-url" => url_for(action: "update", id: object.uuid, controller: object.class.to_s.pluralize.underscore, merge: true),
                     "data-title" => "Set value for #{cwl_shortname(input_schema[:id])}",
                     "data-name" => dn,
                     "data-pk" => "{id: \"#{object.uuid}\", key: \"#{object.class.to_s.underscore}\"}",
                     "data-value" => attrvalue.to_s,
                     # "clear" button interferes with form-control's up/down arrows
                     "data-clear" => false,
                     :class => "editable #{'required' if required} form-control",
                     :id => id
                   }.merge(htmloptions)
    elsif primary_type.is_a? Hash and primary_type[:type] == "enum"
      return link_to attrvalue, '#', {
                     "data-emptytext" => "none",
                     "data-placement" => "bottom",
                     "data-type" => "select",
                     "data-source" => (opt_empty_selection + primary_type[:symbols].map {|i| {:value => cwl_shortname(i), :text => cwl_shortname(i)} }).to_json,
                     "data-url" => url_for(action: "update", id: object.uuid, controller: object.class.to_s.pluralize.underscore, merge: true),
                     "data-title" => "Set value for #{cwl_shortname(input_schema[:id])}",
                     "data-name" => dn,
                     "data-pk" => "{id: \"#{object.uuid}\", key: \"#{object.class.to_s.underscore}\"}",
                     "data-value" => attrvalue,
                     # "clear" button interferes with form-control's up/down arrows
                     "data-clear" => false,
                     :class => "editable #{'required' if required} form-control",
                     :id => id
                   }.merge(htmloptions)
    elsif primary_type.is_a? String
      if ["int", "long"].include? primary_type
        datatype = "number"
      else
        datatype = "text"
      end

      return link_to attrvalue, '#', {
                     "data-emptytext" => "none",
                     "data-placement" => "bottom",
                     "data-type" => datatype,
                     "data-url" => url_for(action: "update", id: object.uuid, controller: object.class.to_s.pluralize.underscore, merge: true),
                     "data-title" => "Set value for #{cwl_shortname(input_schema[:id])}",
                     "data-name" => dn,
                     "data-pk" => "{id: \"#{object.uuid}\", key: \"#{object.class.to_s.underscore}\"}",
                     "data-value" => attrvalue,
                     # "clear" button interferes with form-control's up/down arrows
                     "data-clear" => false,
                     :class => "editable #{'required' if required} form-control",
                     :id => id
                     }.merge(htmloptions)
    else
      return "Unable to render editing control for parameter type #{primary_type}"
    end
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
    "ContainerRequest" => "fa-gears",
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
    "Workflow" => "fa-gears",
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

  def chooser_preview_url_for object, use_preview_selection=false
    case object.class.to_s
    when 'Collection'
      polymorphic_path(object, tab_pane: 'chooser_preview', use_preview_selection: use_preview_selection)
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

  def render_time duration, use_words, round_to_min=true
    render_runtime duration, use_words, round_to_min
  end

  # Keep locators are expected to be of the form \"...<pdh/file_path>\" or \"...<uuid/file_path>\"
  JSON_KEEP_LOCATOR_REGEXP = /([0-9a-f]{32}\+\d+[^'"]*|[a-z0-9]{5}-4zz18-[a-z0-9]{15}[^'"]*)(?=['"]|\z|$)/
  def keep_locator_in_json str
    # Return a list of all matches
    str.scan(JSON_KEEP_LOCATOR_REGEXP).flatten
  end

private
  def is_textile?( object, attr )
    object.textile_attributes.andand.include?(attr)
  end
end
