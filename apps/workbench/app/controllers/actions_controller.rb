class ActionsController < ApplicationController

  @@exposed_actions = {}
  def self.expose_action method, &block
    @@exposed_actions[method] = true
    define_method method, block
  end

  def model_class
    ArvadosBase::resource_class_for_uuid(params[:uuid])
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
    link_selections = Link.filter([['uuid','in',params["selection"]]])
    link_uuids = link_selections.collect(&:uuid)

    # Given a link uuid, we'll add the link's head_uuid. Given another
    # type, we'll add the object itself.
    uuids_to_add = params["selection"] - link_uuids
    uuids_to_add += link_selections.collect(&:head_uuid)

    # Skip anything that's already here.
    already_named = Link.
      filter([['tail_uuid','=',@object.uuid],
              ['head_uuid','in',uuids_to_add],
              ['link_class','=','name']]).
      collect(&:head_uuid)
    uuids_to_add -= already_named

    # Given a name link, we'll try to add the linked object using the
    # same name.
    name_for = {}
    link_selections.
      select { |x| x.link_class == 'name' }.
      each do |link|
      name_for[link.head_uuid] = link.name
    end

    uuids_to_add.each do |s|
      name = name_for[s] || s
      begin
        Link.create(tail_uuid: @object.uuid,
                    head_uuid: s,
                    link_class: 'name',
                    name: name)
      rescue
        Link.create(tail_uuid: @object.uuid,
                    head_uuid: s,
                    link_class: 'name',
                    name: name + " (#{Time.now.localtime})")
      end
    end
    redirect_to @object
  end

  expose_action :combine_selected_files_into_collection do
    lst = []
    files = []
    params["selection"].each do |s|
      m = CollectionsHelper.match(s)
      if m and m[1] and m[2]
        lst.append(m[1] + m[2])
        files.append(m)
      end
    end

    collections = Collection.where(uuid: lst)

    chash = {}
    collections.each do |c|
      c.reload()
      chash[c.uuid] = c
    end

    combined = ""
    files.each do |m|
      mt = chash[m[1]+m[2]].manifest_text
      if m[4]
        IO.popen(['arv-normalize', '--extract', m[4][1..-1]], 'w+b') do |io|
          io.write mt
          io.close_write
          while buf = io.read(2**20)
            combined += buf
          end
        end
      else
        combined += chash[m[1]+m[2]].manifest_text
      end
    end

    normalized = ''
    IO.popen(['arv-normalize'], 'w+b') do |io|
      io.write combined
      io.close_write
      while buf = io.read(2**20)
        normalized += buf
      end
    end

    require 'digest/md5'

    d = Digest::MD5.new()
    d << normalized
    newuuid = "#{d.hexdigest}+#{normalized.length}"

    env = Hash[ENV].
      merge({
              'ARVADOS_API_HOST' =>
              arvados_api_client.arvados_v1_base.
              sub(/\/arvados\/v1/, '').
              sub(/^https?:\/\//, ''),
              'ARVADOS_API_TOKEN' => Thread.current[:arvados_api_token],
              'ARVADOS_API_HOST_INSECURE' =>
              Rails.configuration.arvados_insecure_https ? 'true' : 'false'
            })

    IO.popen([env, 'arv-put', '--raw'], 'w+b') do |io|
      io.write normalized
      io.close_write
      while buf = io.read(2**20)

      end
    end

    newc = Collection.new({:uuid => newuuid, :manifest_text => normalized})
    newc.save!

    chash.each do |k,v|
      l = Link.new({
                     tail_uuid: k,
                     head_uuid: newuuid,
                     link_class: "provenance",
                     name: "provided"
                   })
      l.save!
    end

    redirect_to controller: 'collections', action: :show, id: newc.uuid
  end

end
