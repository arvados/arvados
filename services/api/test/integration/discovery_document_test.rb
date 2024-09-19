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
    # Check committed copies of the discovery document that support code or
    # documentation generation for other Arvados components.
    bad_copies = [
      "sdk/python/arvados-v1-discovery.json",
      "sdk/R/arvados-v1-discovery.json",
    ].filter_map do |rel_path|
      src_path = Rails.root.join("..", "..", rel_path)
      begin
        expected_json = File.open(src_path) { |f| f.read }
      rescue Errno::ENOENT
        expected_json = "(#{src_path} not found)"
      end
      if expected_json == actual_json
        nil
      else
        src_path
      end
    end.to_a
    if bad_copies.any?
      out_path = Rails.root.join("tmp", "test-arvados-v1-discovery.json")
      File.open(out_path, "w") { |f| f.write(actual_json) }
    end
    assert_equal([], bad_copies,
                 "Live discovery document did not match the copies at:\n" +
                 bad_copies.map { |path| " * #{path}\n" }.join("") +
                 "If the live version is correct, copy it to these paths by running:\n" +
                 bad_copies.map { |path| "   cp #{out_path} #{path}\n"}.join(""))
  end

  test "all methods have full descriptions" do
    get "/discovery/v1/apis/arvados/v1/rest"
    assert_response :success
    missing = []
    def missing.check(name, key, spec)
      self << "#{name} #{key}" if spec[key].blank?
    end

    Enumerator::Chain.new(
      *json_response["resources"].map { |_, res| res["methods"].each_value }
    ).each do |method|
      method_name = method["id"]
      missing.check(method_name, "description", method)
      method["parameters"].andand.each_pair do |param_name, param|
        missing.check("#{method_name} #{param_name} parameter", "description", param)
      end
    end

    json_response["schemas"].each_pair do |schema_name, schema|
      missing.check(schema_name, "description", schema)
      schema["properties"].andand.each_pair do |prop_name, prop|
        missing.check("#{schema_name} #{prop_name} property", "description", prop)
      end
    end

    assert_equal(
      missing, [],
      "named methods and schemas are missing documentation",
    )
  end
end
