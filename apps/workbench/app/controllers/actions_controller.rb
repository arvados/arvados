class ActionsController < ApplicationController

  skip_filter :require_thread_api_token, only: [:report_issue_popup, :report_issue]
  skip_filter :check_user_agreements, only: [:report_issue_popup, :report_issue]

  @@exposed_actions = {}
  def self.expose_action method, &block
    @@exposed_actions[method] = true
    define_method method, block
  end

  def model_class
    ArvadosBase::resource_class_for_uuid(params[:uuid])
  end

  def show
    @object = model_class.andand.find(params[:uuid])
    if @object.is_a? Link and
        @object.link_class == 'name' and
        ArvadosBase::resource_class_for_uuid(@object.head_uuid) == Collection
      redirect_to collection_path(id: @object.uuid)
    elsif @object
      redirect_to @object
    else
      raise ActiveRecord::RecordNotFound
    end
  end

  def post
    params.keys.collect(&:to_sym).each do |param|
      if @@exposed_actions[param]
        return self.send(param)
      end
    end
    redirect_to :back
  end

  expose_action :copy_selections_into_project do
    move_or_copy :copy
  end

  expose_action :move_selections_into_project do
    move_or_copy :move
  end

  def move_or_copy action
    uuids_to_add = params["selection"]
    uuids_to_add = [ uuids_to_add ] unless uuids_to_add.is_a? Array
    uuids_to_add.
      collect { |x| ArvadosBase::resource_class_for_uuid(x) }.
      uniq.
      each do |resource_class|
      resource_class.filter([['uuid','in',uuids_to_add]]).each do |src|
        if resource_class == Collection and not Collection.attribute_info.include?(:name)
          dst = Link.new(owner_uuid: @object.uuid,
                         tail_uuid: @object.uuid,
                         head_uuid: src.uuid,
                         link_class: 'name',
                         name: src.uuid)
        else
          case action
          when :copy
            dst = src.dup
            if dst.respond_to? :'name='
              if dst.name
                dst.name = "Copy of #{dst.name}"
              else
                dst.name = "Copy of unnamed #{dst.class_for_display.downcase}"
              end
            end
            if resource_class == Collection
              dst.manifest_text = Collection.select([:manifest_text]).where(uuid: src.uuid).first.manifest_text
            end
          when :move
            dst = src
          else
            raise ArgumentError.new "Unsupported action #{action}"
          end
          dst.owner_uuid = @object.uuid
          dst.tail_uuid = @object.uuid if dst.class == Link
        end
        begin
          dst.save!
        rescue
          dst.name += " (#{Time.now.localtime})" if dst.respond_to? :name=
          dst.save!
        end
      end
    end
    redirect_to @object
  end

  def arv_normalize mt, *opts
    r = ""
    env = Hash[ENV].
      merge({'ARVADOS_API_HOST' =>
              arvados_api_client.arvados_v1_base.
              sub(/\/arvados\/v1/, '').
              sub(/^https?:\/\//, ''),
              'ARVADOS_API_TOKEN' => 'x',
              'ARVADOS_API_HOST_INSECURE' =>
              Rails.configuration.arvados_insecure_https ? 'true' : 'false'
            })
    IO.popen([env, 'arv-normalize'] + opts, 'w+b') do |io|
      io.write mt
      io.close_write
      while buf = io.read(2**16)
        r += buf
      end
    end
    r
  end

  expose_action :combine_selected_files_into_collection do
    uuids = []
    pdhs = []
    files = []
    params["selection"].each do |s|
      a = ArvadosBase::resource_class_for_uuid s
      if a == Link
        begin
          if (m = CollectionsHelper.match(Link.find(s).head_uuid))
            pdhs.append(m[1] + m[2])
            files.append(m)
          end
        rescue
        end
      elsif (m = CollectionsHelper.match(s))
        pdhs.append(m[1] + m[2])
        files.append(m)
      elsif (m = CollectionsHelper.match_uuid_with_optional_filepath(s))
        uuids.append(m[1])
        files.append(m)
      end
    end

    pdhs = pdhs.uniq
    uuids = uuids.uniq
    chash = {}

    Collection.select([:uuid, :manifest_text]).where(uuid: uuids).each do |c|
      chash[c.uuid] = c
    end

    Collection.select([:portable_data_hash, :manifest_text]).where(portable_data_hash: pdhs).each do |c|
      chash[c.portable_data_hash] = c
    end

    combined = ""
    files.each do |m|
      mt = chash[m[1]+m[2]].andand.manifest_text
      if not m[4].nil? and m[4].size > 1
        combined += arv_normalize mt, '--extract', m[4][1..-1]
      else
        combined += mt
      end
    end

    normalized = arv_normalize combined
    newc = Collection.new({:manifest_text => normalized})
    newc.name = newc.name || "Collection created at #{Time.now.localtime}"

    # set owner_uuid to current project, provided it is writable
    current_project_writable = false
    action_data = JSON.parse(params['action_data']) if params['action_data']
    if action_data && action_data['current_project_uuid']
      current_project = Group.find(action_data['current_project_uuid']) rescue nil
      if (current_project && current_project.writable_by.andand.include?(current_user.uuid))
        newc.owner_uuid = action_data['current_project_uuid']
        current_project_writable = true
      end
    end

    newc.save!

    chash.each do |k,v|
      l = Link.new({
                     tail_uuid: k,
                     head_uuid: newc.uuid,
                     link_class: "provenance",
                     name: "provided"
                   })
      l.save!
    end

    msg = current_project_writable ?
              "Created new collection in the project #{current_project.name}." :
              "Created new collection in your Home project."

    redirect_to newc, flash: {'message' => msg}
  end

  def report_issue_popup
    respond_to do |format|
      format.js
      format.html
    end
  end

  def report_issue
    logger.warn "report_issue: #{params.inspect}"

    respond_to do |format|
      IssueReporter.send_report(current_user, params).deliver
      format.js {render nothing: true}
    end
  end

end
