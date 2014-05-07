class Arvados::V1::GroupsController < ApplicationController

  def self._contents_requires_parameters
    _index_requires_parameters.
      merge({
              include_linked: {
                type: 'boolean', required: false, default: false
              },
            })
  end

  def contents
    all_objects = []
    all_available = 0

    # Trick apply_where_limit_order_params into applying suitable
    # per-table values. *_all are the real ones we'll apply to the
    # aggregate set.
    limit_all = @limit
    offset_all = @offset
    @orders = []

    ArvadosModel.descendants.reject(&:abstract_class?).sort_by(&:to_s).
      each do |klass|
      case klass.to_s
        # We might expect klass==Link etc. here, but we would be
        # disappointed: when Rails reloads model classes, we get two
        # distinct classes called Link which do not equal each
        # other. But we can still rely on klass.to_s to be "Link".
      when 'ApiClientAuthorization', 'UserAgreement', 'Link'
        # Do not want.
      else
        @objects = klass.readable_by(*@read_users)
        cond_sql = "#{klass.table_name}.owner_uuid = ?"
        cond_params = [@object.uuid]
        if params[:include_linked]
          cond_sql += " OR #{klass.table_name}.uuid IN (SELECT head_uuid FROM links WHERE link_class=#{klass.sanitize 'name'} AND links.tail_uuid=#{klass.sanitize @object.uuid})"
        end
        @objects = @objects.where(cond_sql, *cond_params).order("#{klass.table_name}.uuid")
        @limit = limit_all - all_objects.count
        apply_where_limit_order_params
        items_available = @objects.
          except(:limit).except(:offset).
          count(:id, distinct: true)
        all_available += items_available
        @offset = [@offset - items_available, 0].max

        all_objects += @objects.to_a
      end
    end
    @objects = all_objects || []
    @links = Link.where('link_class=? and tail_uuid=?'\
                        ' and head_uuid in (?)',
                        'name',
                        @object.uuid,
                        @objects.collect(&:uuid))
    @object_list = {
      :kind  => "arvados#objectList",
      :etag => "",
      :self_link => "",
      :links => @links.as_api_response(nil),
      :offset => offset_all,
      :limit => limit_all,
      :items_available => all_available,
      :items => @objects.as_api_response(nil)
    }
    render json: @object_list
  end

end
