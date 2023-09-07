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
    actual_json = JSON.pretty_generate(canonical)

    # Currently the Python SDK is the only component using this copy of the
    # discovery document, and storing it with the source simplifies the build
    # process, so it lives there. If another component wants to use it later,
    # we might consider moving it to a more general subdirectory, but then the
    # Python build process will need to be extended to accommodate that.
    src_path = Rails.root.join("../../sdk/python/arvados-v1-discovery.json")
    begin
      expected_json = File.open(src_path) { |f| f.read }
    rescue Errno::ENOENT
      expected_json = "(#{src_path} not found)"
    end

    out_path = Rails.root.join("tmp", "test-arvados-v1-discovery.json")
    if expected_json != actual_json
      File.open(out_path, "w") { |f| f.write(actual_json) }
    end
    assert_equal(expected_json, actual_json, [
                   "#{src_path} did not match the live discovery document",
                   "Current live version saved to #{out_path}",
                   "Commit that to #{src_path} to regenerate documentation",
                 ].join(". "))
  end
end
