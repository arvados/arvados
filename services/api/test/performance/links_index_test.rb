require 'test_helper'
require 'rails/performance_test_help'

class IndexTest < ActionDispatch::PerformanceTest
  def test_links_index
    get '/arvados/v1/links', {format: :json}, auth(:admin)
  end
  def test_links_index_with_filters
    get '/arvados/v1/links', {format: :json, filters: [%w[head_uuid is_a arvados#collection]].to_json}, auth(:admin)
  end
  def test_collections_index
    get '/arvados/v1/collections', {format: :json}, auth(:admin)
  end
end
