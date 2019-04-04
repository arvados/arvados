# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'benchmark'

class IndexTest < ActionDispatch::IntegrationTest
  def test_links_index
    puts("Get links index: ", Benchmark.measure do
      get '/arvados/v1/links', params: {
        limit: 1000,
        format: :json
      }, headers: auth(:admin)
    end)
  end
  def test_links_index_with_filters
    puts("Get links index with filters: ", Benchmark.measure do
      get '/arvados/v1/links', params: {
        format: :json,
        filters: [%w[head_uuid is_a arvados#collection]].to_json
      }, headers: auth(:admin)
    end)
  end
  def test_collections_index
    puts("Get collections index: ", Benchmark.measure do
      get '/arvados/v1/collections', params: {
        format: :json
        }, headers: auth(:admin)
      end)
  end
end
