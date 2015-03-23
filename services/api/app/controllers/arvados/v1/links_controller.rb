class Arvados::V1::LinksController < ApplicationController

  def check_uuid_kind uuid, kind
    if kind and ArvadosModel::resource_class_for_uuid(uuid).andand.kind != kind
      send_error("'#{kind}' does not match uuid '#{uuid}', expected '#{ArvadosModel::resource_class_for_uuid(uuid).andand.kind}'",
                 status: 422)
      nil
    else
      true
    end
  end

  def create
    return if ! check_uuid_kind resource_attrs[:head_uuid], resource_attrs[:head_kind]
    return if ! check_uuid_kind resource_attrs[:tail_uuid], resource_attrs[:tail_kind]

    resource_attrs.delete :head_kind
    resource_attrs.delete :tail_kind
    super
  end

  def get_permissions
    if current_user.andand.can?(manage: @object)
      # find all links and return them
      @objects = Link.where(link_class: "permission",
                            head_uuid: params[:uuid])
      @offset = 0
      @limit = @objects.count
      render_list
    else
      render :json => { errors: ['Forbidden'] }.to_json, status: 403
    end
  end

  protected

  def find_object_by_uuid
    if action_name == 'get_permissions'
      # get_permissions accepts a UUID for any kind of object.
      @object = ArvadosModel::resource_class_for_uuid(params[:uuid])
        .readable_by(*@read_users)
        .where(uuid: params[:uuid])
        .first
    else
      super
      if @object.nil?
        # Normally group permission links are not readable_by users.
        # Make an exception for users with permission to manage the group.
        # FIXME: Solve this more generally - see the controller tests.
        link = Link.find_by_uuid(params[:uuid])
        if (not link.nil?) and
            (link.link_class == "permission") and
            (@read_users.any? { |u| u.can?(manage: link.head_uuid) })
          @object = link
        end
      end
    end
  end

  # Overrides ApplicationController load_where_param
  def load_where_param
    super

    # head_kind and tail_kind columns are now virtual,
    # equivilent functionality is now provided by
    # 'is_a', so fix up any old-style 'where' clauses.
    if @where
      @filters ||= []
      if @where[:head_kind]
        @filters << ['head_uuid', 'is_a', @where[:head_kind]]
        @where.delete :head_kind
      end
      if @where[:tail_kind]
        @filters << ['tail_uuid', 'is_a', @where[:tail_kind]]
        @where.delete :tail_kind
      end
    end
  end

  # Overrides ApplicationController load_filters_param
  def load_filters_param
    super

    # head_kind and tail_kind columns are now virtual,
    # equivilent functionality is now provided by
    # 'is_a', so fix up any old-style 'filter' clauses.
    @filters = @filters.map do |k|
      if k[0] == 'head_kind' and k[1] == '='
        ['head_uuid', 'is_a', k[2]]
      elsif k[0] == 'tail_kind' and k[1] == '='
        ['tail_uuid', 'is_a', k[2]]
      else
        k
      end
    end
  end

end
