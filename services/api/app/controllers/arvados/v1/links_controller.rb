class Arvados::V1::LinksController < ApplicationController

  def create
    resource_attrs.delete :head_kind
    resource_attrs.delete :tail_kind
    super
  end

  protected

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
