require "arvados/keep"

class Arvados::V1::CollectionsController < ApplicationController
  def create
    if resource_attrs[:uuid] and (loc = Keep::Locator.parse(resource_attrs[:uuid]))
      resource_attrs[:portable_data_hash] = loc.to_s
      resource_attrs.delete :uuid
    end
    super
  end

  def find_object_by_uuid
    if loc = Keep::Locator.parse(params[:id])
      loc.strip_hints!
      if c = Collection.readable_by(*@read_users).where({ portable_data_hash: loc.to_s }).limit(1).first
        @object = {
          uuid: c.portable_data_hash,
          portable_data_hash: c.portable_data_hash,
          manifest_text: c.signed_manifest_text,
        }
      end
    else
      super
    end
    true
  end

  def show
    if @object.is_a? Collection
      super
    else
      send_json @object
    end
  end

  def find_collections(visited, sp, &b)
    case sp
    when ArvadosModel
      sp.class.columns.each do |c|
        find_collections(visited, sp[c.name.to_sym], &b) if c.name != "log"
      end
    when Hash
      sp.each do |k, v|
        find_collections(visited, v, &b)
      end
    when Array
      sp.each do |v|
        find_collections(visited, v, &b)
      end
    when String
      if m = /[a-f0-9]{32}\+\d+/.match(sp)
        yield m[0], nil
      elsif m = Collection.uuid_regex.match(sp)
        yield nil, m[0]
      end
    end
  end

  def search_edges(visited, uuid, direction)
    if uuid.nil? or uuid.empty? or visited[uuid]
      return
    end

    if loc = Keep::Locator.parse(uuid)
      loc.strip_hints!
      return if visited[loc.to_s]
    end

    logger.debug "visiting #{uuid}"

    if loc
      # uuid is a portable_data_hash
      collections = Collection.readable_by(*@read_users).where(portable_data_hash: loc.to_s)
      c = collections.limit(2).all
      if c.size == 1
        visited[loc.to_s] = c[0]
      elsif c.size > 1
        name = collections.limit(1).where("name <> ''").first
        if name
          visited[loc.to_s] = {
            portable_data_hash: c[0].portable_data_hash,
            name: "#{name.name} + #{collections.count-1} more"
          }
        else
          visited[loc.to_s] = {
            portable_data_hash: c[0].portable_data_hash,
            name: loc.to_s
          }
        end
      end

      if direction == :search_up
        # Search upstream for jobs where this locator is the output of some job
        Job.readable_by(*@read_users).where(output: loc.to_s).each do |job|
          search_edges(visited, job.uuid, :search_up)
        end

        Job.readable_by(*@read_users).where(log: loc.to_s).each do |job|
          search_edges(visited, job.uuid, :search_up)
        end
      elsif direction == :search_down
        if loc.to_s == "d41d8cd98f00b204e9800998ecf8427e+0"
          # Special case, don't follow the empty collection.
          return
        end

        # Search downstream for jobs where this locator is in script_parameters
        Job.readable_by(*@read_users).where(["jobs.script_parameters like ?", "%#{loc.to_s}%"]).each do |job|
          search_edges(visited, job.uuid, :search_down)
        end

        Job.readable_by(*@read_users).where(["jobs.docker_image_locator = ?", "#{loc.to_s}"]).each do |job|
          search_edges(visited, job.uuid, :search_down)
        end
      end
    else
      # uuid is a regular Arvados UUID
      rsc = ArvadosModel::resource_class_for_uuid uuid
      if rsc == Job
        Job.readable_by(*@read_users).where(uuid: uuid).each do |job|
          visited[uuid] = job.as_api_response
          if direction == :search_up
            # Follow upstream collections referenced in the script parameters
            find_collections(visited, job) do |hash, uuid|
              search_edges(visited, hash, :search_up) if hash
              search_edges(visited, uuid, :search_up) if uuid
            end
          elsif direction == :search_down
            # Follow downstream job output
            search_edges(visited, job.output, direction)
          end
        end
      elsif rsc == Collection
        if c = Collection.readable_by(*@read_users).where(uuid: uuid).limit(1).first
          search_edges(visited, c.portable_data_hash, direction)
          visited[c.portable_data_hash] = c.as_api_response
        end
      elsif rsc != nil
        rsc.where(uuid: uuid).each do |r|
          visited[uuid] = r.as_api_response
        end
      end
    end

    if direction == :search_up
      # Search for provenance links pointing to the current uuid
      Link.readable_by(*@read_users).
        where(head_uuid: uuid, link_class: "provenance").
        each do |link|
        visited[link.uuid] = link.as_api_response
        search_edges(visited, link.tail_uuid, direction)
      end
    elsif direction == :search_down
      # Search for provenance links emanating from the current uuid
      Link.readable_by(current_user).
        where(tail_uuid: uuid, link_class: "provenance").
        each do |link|
        visited[link.uuid] = link.as_api_response
        search_edges(visited, link.head_uuid, direction)
      end
    end
  end

  def provenance
    visited = {}
    search_edges(visited, @object[:portable_data_hash], :search_up)
    search_edges(visited, @object[:uuid], :search_up)
    send_json visited
  end

  def used_by
    visited = {}
    search_edges(visited, @object[:uuid], :search_down)
    search_edges(visited, @object[:portable_data_hash], :search_down)
    send_json visited
  end

  protected

  def load_limit_offset_order_params *args
    if action_name == 'index'
      # Omit manifest_text from index results unless expressly selected.
      @select ||= model_class.selectable_attributes - ["manifest_text"]
    end
    super
  end
end
