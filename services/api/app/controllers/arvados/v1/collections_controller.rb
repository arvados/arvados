class Arvados::V1::CollectionsController < ApplicationController
  def create
    if !resource_attrs[:manifest_text]
      return send_error("'manifest_text' attribute must be specified",
                        status: :unprocessable_entity)
    end

    # Check permissions on the collection manifest.
    # If any signature cannot be verified, return 403 Permission denied.
    api_token = current_api_client_authorization.andand.api_token
    signing_opts = {
      key: Rails.configuration.blob_signing_key,
      api_token: api_token,
      ttl: Rails.configuration.blob_signing_ttl,
    }
    resource_attrs[:manifest_text].lines.each do |entry|
      entry.split[1..-1].each do |tok|
        if /^[[:digit:]]+:[[:digit:]]+:/.match tok
          # This is a filename token, not a blob locator. Note that we
          # keep checking tokens after this, even though manifest
          # format dictates that all subsequent tokens will also be
          # filenames. Safety first!
        elsif Blob.verify_signature tok, signing_opts
          # OK.
        elsif Locator.parse(tok).andand.signature
          # Signature provided, but verify_signature did not like it.
          logger.warn "Invalid signature on locator #{tok}"
          raise ArvadosModel::PermissionDeniedError
        elsif Rails.configuration.permit_create_collection_with_unsigned_manifest
          # No signature provided, but we are running in insecure mode.
          logger.debug "Missing signature on locator #{tok} ignored"
        elsif Blob.new(tok).empty?
          # No signature provided -- but no data to protect, either.
        else
          logger.warn "Missing signature on locator #{tok}"
          raise ArvadosModel::PermissionDeniedError
        end
      end
    end

    # Remove any permission signatures from the manifest.
    resource_attrs[:manifest_text]
      .gsub!(/ [[:xdigit:]]{32}(\+[[:digit:]]+)?(\+\S+)/) { |word|
      word.strip!
      loc = Locator.parse(word)
      if loc
        " " + loc.without_signature.to_s
      else
        " " + word
      end
    }

    super
  end

  def find_object_by_uuid
    if loc = Locator.parse(params[:id])
      loc.strip_hints!
      if c = Collection.readable_by(*@read_users).where({ portable_data_hash: loc.to_s }).limit(1).first
        @object = {
          portable_data_hash: c.portable_data_hash,
          manifest_text: c.manifest_text,
          files: c.files,
          data_size: c.data_size
        }
      end
    else
      super
    end
    true
  end

  def show
    if current_api_client_authorization
      signing_opts = {
        key: Rails.configuration.blob_signing_key,
        api_token: current_api_client_authorization.api_token,
        ttl: Rails.configuration.blob_signing_ttl,
      }
      @object[:manifest_text]
        .gsub!(/ [[:xdigit:]]{32}(\+[[:digit:]]+)?(\+\S+)/) { |word|
        word.strip!
        loc = Locator.parse(word)
        if loc
          " " + Blob.sign_locator(word, signing_opts)
        else
          " " + word
        end
      }
    end
    if @object.is_a? Collection
      render json: @object.as_api_response(:with_data)
    else
      render json: @object
    end
  end

  def script_param_edges(visited, sp)
    case sp
    when Hash
      sp.each do |k, v|
        script_param_edges(visited, v)
      end
    when Array
      sp.each do |v|
        script_param_edges(visited, v)
      end
    when String
      return if sp.empty?
      if loc = Locator.parse(sp)
        search_edges(visited, loc.to_s, :search_up)
      end
    end
  end

  def search_edges(visited, uuid, direction)
    if uuid.nil? or uuid.empty? or visited[uuid]
      return
    end

    if loc = Locator.parse(uuid)
      loc.strip_hints!
      return if visited[loc.to_s]
    end

    logger.debug "visiting #{uuid}"

    if loc
      # uuid is a portable_data_hash
      if c = Collection.readable_by(*@read_users).where(portable_data_hash: loc.to_s).limit(1).first
        visited[loc.to_s] = {
          portable_data_hash: c.portable_data_hash,
          files: c.files,
          data_size: c.data_size
        }
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
      end
    else
      # uuid is a regular Arvados UUID
      rsc = ArvadosModel::resource_class_for_uuid uuid
      if rsc == Job
        Job.readable_by(*@read_users).where(uuid: uuid).each do |job|
          visited[uuid] = job.as_api_response
          if direction == :search_up
            # Follow upstream collections referenced in the script parameters
            script_param_edges(visited, job.script_parameters)
          elsif direction == :search_down
            # Follow downstream job output
            search_edges(visited, job.output, direction)
          end
        end
      elsif rsc == Collection
        if c = Collection.readable_by(*@read_users).where(uuid: uuid).limit(1).first
          visited[uuid] = c.as_api_response
          search_edges(visited, c.portable_data_hash, direction)
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
    search_edges(visited, @object[:uuid] || @object[:portable_data_hash], :search_up)
    render json: visited
  end

  def used_by
    visited = {}
    search_edges(visited, @object[:uuid] || @object[:portable_data_hash], :search_down)
    render json: visited
  end

end
