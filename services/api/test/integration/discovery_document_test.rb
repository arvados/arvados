# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class DiscoveryDocumentTest < ActionDispatch::IntegrationTest
  CANONICAL_FIELDS = [
    "auth",
    "basePath",
    "batchPath",
    "description",
    "discoveryVersion",
    "documentationLink",
    "id",
    "kind",
    "name",
    "parameters",
    "protocol",
    "resources",
    "revision",
    "schemas",
    "servicePath",
    "title",
    "version",
  ]

  test "canonical discovery document is saved to checkout" do
    get "/discovery/v1/apis/arvados/v1/rest"
    assert_response :success
    canonical = Hash[CANONICAL_FIELDS.map { |key| [key, json_response[key]] }]
    missing = canonical.select { |key| canonical[key].nil? }
    assert(missing.empty?, "discovery document missing required fields")

    expected = JSON.pretty_generate(canonical)
    src_path = Rails.root.join("../../doc/arvados-v1-discovery.json")
    begin
      actual = File.open(src_path) { |f| f.read }
    rescue Errno::ENOENT
      actual = "(#{src_path} not found)"
    end

    out_path = Rails.root.join("tmp", "test-arvados-v1-discovery.json")
    if expected != actual
      File.open(out_path, "w") { |f| f.write(expected) }
    end
    assert_equal(expected, actual, [
                   "#{src_path} did not match the live discovery document",
                   "Current live version saved to #{out_path}",
                   "Commit that to #{src_path} to regenerate documentation",
                 ].join(". "))
  end
end
