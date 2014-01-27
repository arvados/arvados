class Arvados::V1::CollectionsController < ApplicationController
  def create
    # Collections are owned by system_user. Creating a collection has
    # two effects: The collection is added if it doesn't already
    # exist, and a "permission" Link is added (if one doesn't already
    # exist) giving the current user (or specified owner_uuid)
    # permission to read it.
    owner_uuid = resource_attrs.delete(:owner_uuid) || current_user.uuid
    owner_kind = if owner_uuid.match(/-(\w+)-/)[1] == User.uuid_prefix
                   'arvados#user'
                 else
                   'arvados#group'
                 end
    unless current_user.can? write: owner_uuid
      raise ArvadosModel::PermissionDeniedError
    end
    act_as_system_user do
      @object = model_class.new resource_attrs.reject { |k,v| k == :owner_uuid }
      begin
        @object.save!
      rescue ActiveRecord::RecordNotUnique
        logger.debug resource_attrs.inspect
        if resource_attrs[:manifest_text] and resource_attrs[:uuid]
          @existing_object = model_class.
            where('uuid=? and manifest_text=?',
                  resource_attrs[:uuid],
                  resource_attrs[:manifest_text]).
            first
          @object = @existing_object || @object
        end
      end

      if @object
        link_attrs = {
          owner_uuid: owner_uuid,
          link_class: 'permission',
          name: 'can_read',
          head_kind: 'arvados#collection',
          head_uuid: @object.uuid,
          tail_kind: owner_kind,
          tail_uuid: owner_uuid
        }
        ActiveRecord::Base.transaction do
          if Link.where(link_attrs).empty?
            Link.create! link_attrs
          end
        end
      end
    end
    show
  end

  def collection_uuid(uuid)
    m = /([a-f0-9]{32}(\+[0-9]+)?)(\+.*)?/.match(uuid)
    if m
      m[1]
    else
      nil
    end
  end

  def script_param_edges(visited, sp)
    if sp and not sp.empty?
      case sp
      when Hash
        sp.each do |k, v|
          script_param_edges(visited, v)
        end
      when Array
        sp.each do |v|
          script_param_edges(visited, v)
        end
      else
        m = collection_uuid(sp)
        if m
          generate_provenance_edges(visited, m)
        end
      end
    end
    gr
  end

  def generate_provenance_edges(visited, uuid)
    m = collection_uuid(uuid)

    if not uuid or uuid.empty? or visited[uuid] or visited[m]
      return ""
    end

    #puts "visiting #{uuid}"

    if m  
      # uuid is a collection
      uuid = m
      Collection.where(uuid:"uuid").each do |c|
        visited[uuid] = c
      end

      Job.where(output: uuid).each do |job|
        generate_provenance_edges(visited, job.uuid)
      end

      Job.where(log: uuid).each do |job|
        generate_provenance_edges(visited, job.uuid)
      end
      
    else
      visited[uuid] = true

      # uuid is something else
      rsc = ArvadosBase::resource_class_for_uuid uuid

      if rsc == Job
        Job.where(uuid: uuid).each do |job|
          visited[uuid] = job
          script_param_edges(visited, job, "", job.script_parameters)
        end
    end

    Link.where(head_uuid: uuid, link_class: "provenance").each do |link|
      generate_provenance_edges(visited, link.tail_uuid)
    end

    #puts "finished #{uuid}"

    gr
  end

  def provenance 
    visited = {}
    generate_provenance_edges(visited, @object.uuid)
    visited
  end

  protected

  def find_object_by_uuid
    super
    if !@object and !params[:uuid].match(/^[0-9a-f]+\+\d+$/)
      # Normalize the given uuid and search again.
      hash_part = params[:uuid].match(/^([0-9a-f]*)/)[1]
      collection = Collection.where('uuid like ?', hash_part + '+%').first
      if collection
        # We know the collection exists, and what its real uuid is in
        # the database. Now, throw out @objects and repeat the usual
        # lookup procedure. (Returning the collection at this point
        # would bypass permission checks.)
        @objects = nil
        @where = { uuid: collection.uuid }
        find_objects_for_index
        @object = @objects.first
      end
    end
  end
end
