class ApiClientAuthorizationsController < ApplicationController
  def index
    m = model_class.all
    items_available = m.items_available
    offset = m.result_offset
    limit = m.result_limit
    filtered = m.to_ary.reject do |x|
      x.api_client_id == 0 or (x.expires_at and x.expires_at < Time.now) rescue false
    end
    ArvadosApiClient::patch_paging_vars(filtered, items_available, offset, limit, nil)
    @objects = ArvadosResourceList.new(ApiClientAuthorization)
    @objects.results= filtered
    super
  end

  def index_pane_list
    %w(Recent Help)
  end

end
