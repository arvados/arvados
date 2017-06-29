# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::LogsController < ApplicationController
  # Overrides ApplicationController load_where_param
  def load_where_param
    super

    # object_kind and column is now virtual,
    # equivilent functionality is now provided by
    # 'is_a', so fix up any old-style 'where' clauses.
    if @where
      @filters ||= []
      if @where[:object_kind]
        @filters << ['object_uuid', 'is_a', @where[:object_kind]]
        @where.delete :object_kind
      end
    end
  end

  # Overrides ApplicationController load_filters_param
  def load_filters_param
    super

    # object_kind and column is now virtual,
    # equivilent functionality is now provided by
    # 'is_a', so fix up any old-style 'filter' clauses.
    @filters = @filters.map do |k|
      if k[0] == 'object_kind' and k[1] == '='
        ['object_uuid', 'is_a', k[2]]
      else
        k
      end
    end
  end

end
