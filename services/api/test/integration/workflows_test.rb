# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class WorkflowsApiTest < ActionDispatch::IntegrationTest
  fixtures :all

  def create_workflow_collection_helper
    post "/arvados/v1/collections",
         params: {:format => :json,
                  collection: {
                    name: "test workflow",
                    description: "the workflow that tests linking collection and workflow records",
                    properties: {
                      "type": "workflow",
                                 "arv:workflowMain": "foo.cwl",
                                 "arv:cwl_inputs": [{
                                                      "id": "#main/x",
                                                     "type": "int",
                                                    }
                                                   ],
                                 "arv:cwl_outputs": [{
                                                      "id": "#main/y",
                                                     "type": "File",
                                                     }],
                                 "arv:cwl_requirements": [
                                                         ],
                                 "arv:cwl_hints": [
                                                  ],
                    }
                  }
                 },
         headers: auth(:active),
         as: :json
    assert_response :success
    json_response
  end

  test "link a workflow to a collection" do

    collection_response = create_workflow_collection_helper
    assert_equal(collection_response["name"], "test workflow")
    assert_equal(collection_response["description"], "the workflow that tests linking collection and workflow records")
    assert_equal(collection_response["owner_uuid"], users(:active).uuid)

    # Now create a workflow linked to the collection.
    post "/arvados/v1/workflows",
         params: {:format => :json,
                  :workflow => {
                    collection_uuid: collection_response["uuid"]
                  }
                 },
      headers: auth(:active)
    assert_response :success
    workflow_response = json_response
    assert_equal(collection_response["name"], workflow_response["name"])
    assert_equal(collection_response["description"], workflow_response["description"])
    assert_equal(collection_response["owner_uuid"], workflow_response["owner_uuid"])
    assert_equal({"cwlVersion"=>"v1.2",
                  "$graph"=>[
                    {"class"=>"Workflow",
                     "id"=>"#main",
                     "inputs"=>[{"id"=>"#main/x", "type"=>"int"}],
                     "outputs"=>[{"id"=>"#main/y", "type"=>"File", "outputSource"=>"#main/step/y"}],
                     "steps"=>[{"id"=>"#main/foo.cwl",
                                "in"=>[{"id"=>"#main/step/x", "source"=>"#main/x"}],
                                "out"=>[{"id"=>"#main/step/y"}],
                                "run"=>"keep:d41d8cd98f00b204e9800998ecf8427e+0/foo.cwl",
                                "label"=>"test workflow"}],
                     "requirements"=>[{"class"=>"SubworkflowFeatureRequirement"}],
                     "hints"=>[]}]},
                 JSON.parse(workflow_response["definition"]))

    # Now update the collection and check that the linked workflow record was also updated.
    patch "/arvados/v1/collections/#{collection_response['uuid']}",
         params: {:format => :json,
                  collection: {
                    name: "test workflow v2",
                    description: "the second version of the workflow that tests linking collection and workflow records",
                    owner_uuid: groups(:private).uuid,
                    properties: {
                      "type": "workflow",
                                 "arv:workflowMain": "foo.cwl",
                                 "arv:cwl_inputs": [{
                                                      "id": "#main/w",
                                                     "type": "int",
                                                    },
                                                    {
                                                      "id": "#main/x",
                                                     "type": "int",
                                                    }
                                                   ],
                                 "arv:cwl_outputs": [{
                                                      "id": "#main/y",
                                                     "type": "File",
                                                     },
                                                    {
                                                      "id": "#main/z",
                                                     "type": "File",
                                                     }],
                                 "arv:cwl_requirements": [
                                                         ],
                                 "arv:cwl_hints": [
                                                  ],
                    }
                  }
                 },
         headers: auth(:active),
         as: :json
    assert_response :success
    collection_response = json_response
    assert_equal(collection_response["name"], "test workflow v2")
    assert_equal(collection_response["description"], "the second version of the workflow that tests linking collection and workflow records")
    assert_equal(collection_response["owner_uuid"], groups(:private).uuid)

    get "/arvados/v1/workflows/#{workflow_response['uuid']}", headers: auth(:active)
    assert_response :success
    workflow_response = json_response
    assert_equal(collection_response["name"], workflow_response["name"])
    assert_equal(collection_response["description"], workflow_response["description"])
    assert_equal(collection_response["owner_uuid"], workflow_response["owner_uuid"])
    assert_equal({"cwlVersion"=>"v1.2",
                  "$graph"=>[
                    {"class"=>"Workflow",
                     "id"=>"#main",
                     "inputs"=>[{"id"=>"#main/w", "type"=>"int"},
                                {"id"=>"#main/x", "type"=>"int"}
                               ],
                     "outputs"=>[{"id"=>"#main/y", "type"=>"File", "outputSource"=>"#main/step/y"},
                                 {"id"=>"#main/z", "type"=>"File", "outputSource"=>"#main/step/z"}],
                     "steps"=>[{"id"=>"#main/foo.cwl",
                                "in"=>[{"id"=>"#main/step/w", "source"=>"#main/w"},
                                       {"id"=>"#main/step/x", "source"=>"#main/x"}],
                                "out"=>[{"id"=>"#main/step/y"}, {"id"=>"#main/step/z"}],
                                "run"=>"keep:d41d8cd98f00b204e9800998ecf8427e+0/foo.cwl",
                                "label"=>"test workflow v2"}],
                     "requirements"=>[{"class"=>"SubworkflowFeatureRequirement"}],
                     "hints"=>[]}]},

                 JSON.parse(workflow_response["definition"]))
  end

  test "workflow cannot be modified after it is linked" do
    # Now create a workflow linked to the collection.
    post "/arvados/v1/workflows",
         params: {:format => :json,
                  :workflow => {
                    name: "legacy"
                  }
                 },
      headers: auth(:active)
    assert_response :success
    workflow_response = json_response
    assert_equal("legacy", workflow_response["name"])

    patch "/arvados/v1/workflows/#{workflow_response['uuid']}",
         params: {:format => :json,
                  :workflow => {
                    name: "legacy v2"
                  }
                 },
         headers: auth(:active),
         as: :json
    assert_response :success
    workflow_response = json_response
    assert_equal("legacy v2", workflow_response["name"])

    collection_response = create_workflow_collection_helper
    patch "/arvados/v1/workflows/#{workflow_response['uuid']}",
         params: {:format => :json,
                  :workflow => {
                    collection_uuid: collection_response['uuid']
                  }
                 },
         headers: auth(:active),
         as: :json
    assert_response :success
    workflow_response = json_response
    assert_equal(collection_response['name'], workflow_response["name"])

    patch "/arvados/v1/workflows/#{workflow_response['uuid']}",
         params: {:format => :json,
                  :workflow => {
                    name: "legacy v2"
                  }
                 },
         headers: auth(:active),
         as: :json
    assert_response 403

  end

  test "trashing collection also hides workflow" do

    collection_response = create_workflow_collection_helper

    # Now create a workflow linked to the collection.
    post "/arvados/v1/workflows",
         params: {:format => :json,
                  :workflow => {
                    collection_uuid: collection_response["uuid"]
                  }
                 },
      headers: auth(:active)
    assert_response :success
    workflow_response = json_response

    get "/arvados/v1/workflows/#{workflow_response['uuid']}", headers: auth(:active)
    assert_response :success

    # Now trash the collection
    post "/arvados/v1/collections/#{collection_response['uuid']}/trash", headers: auth(:active)
    assert_response :success

    get "/arvados/v1/collections/#{collection_response['uuid']}", headers: auth(:active)
    assert_response 404

    get "/arvados/v1/workflows/#{workflow_response['uuid']}", headers: auth(:active)
    assert_response 404

    # Now untrash the collection
    post "/arvados/v1/collections/#{collection_response['uuid']}/untrash", headers: auth(:active)
    assert_response :success

    get "/arvados/v1/collections/#{collection_response['uuid']}", headers: auth(:active)
    assert_response :success

    get "/arvados/v1/workflows/#{workflow_response['uuid']}", headers: auth(:active)
    assert_response :success
  end

  test "collection is missing cwl_inputs" do
    post "/arvados/v1/collections",
         params: {:format => :json,
                  collection: {
                    name: "test workflow",
                    description: "the workflow that tests linking collection and workflow records",
                    properties: {
                      "type": "workflow",
                                 "arv:workflowMain": "foo.cwl"
                    }
                  }
                 },
         headers: auth(:active),
         as: :json
    assert_response :success
    collection_response = json_response

    post "/arvados/v1/workflows",
         params: {:format => :json,
                  :workflow => {
                    collection_uuid: collection_response["uuid"]
                  }
                 },
      headers: auth(:active)
    assert_response 422
  end

  test "collection cwl_inputs wrong type" do
    post "/arvados/v1/collections",
         params: {:format => :json,
                  collection: {
                    name: "test workflow",
                    description: "the workflow that tests linking collection and workflow records",
                    properties: {
                      "type": "workflow",
                                 "arv:workflowMain": "foo.cwl",
                                 "arv:cwl_inputs": { "#main/x": {
                                                                  "type": "int"
                                                                }
                                                   },
                                 "arv:cwl_outputs": [{
                                                      "id": "#main/y",
                                                     "type": "File",
                                                     }],
                                 "arv:cwl_requirements": [
                                                         ],
                                 "arv:cwl_hints": [
                                                  ],

                    }
                  }
                 },
         headers: auth(:active),
         as: :json
    assert_response :success
    collection_response = json_response

    post "/arvados/v1/workflows",
         params: {:format => :json,
                  :workflow => {
                    collection_uuid: collection_response["uuid"]
                  }
                 },
      headers: auth(:active)
    assert_response 422
  end

  test "destroying collection destroys linked workflow" do
    collection_response = create_workflow_collection_helper

    # Now create a workflow linked to the collection.
    post "/arvados/v1/workflows",
         params: {:format => :json,
                  :workflow => {
                    collection_uuid: collection_response["uuid"]
                  }
                 },
      headers: auth(:active)
    assert_response :success
    workflow_response = json_response

    assert_not_nil Collection.find_by_uuid(collection_response['uuid'])
    assert_not_nil Workflow.find_by_uuid(workflow_response['uuid'])

    delete "/arvados/v1/workflows/#{workflow_response['uuid']}",
         params: {:format => :json},
         headers: auth(:active)
    assert_response :success
    workflow_response = json_response

    assert_not_nil Collection.find_by_uuid(collection_response['uuid'])
    assert_nil Workflow.find_by_uuid(workflow_response['uuid'])
  end

  test "workflow can be deleted without deleting collection" do
    collection_response = create_workflow_collection_helper

    # Now create a workflow linked to the collection.
    post "/arvados/v1/workflows",
         params: {:format => :json,
                  :workflow => {
                    collection_uuid: collection_response["uuid"]
                  }
                 },
      headers: auth(:active)
    assert_response :success
    workflow_response = json_response

    assert_not_nil Collection.find_by_uuid(collection_response['uuid'])
    assert_not_nil Workflow.find_by_uuid(workflow_response['uuid'])

    Collection.find_by_uuid(collection_response['uuid']).destroy

    assert_nil Collection.find_by_uuid(collection_response['uuid'])
    assert_nil Workflow.find_by_uuid(workflow_response['uuid'])
  end

  test "group contents endpoint supports include=collection_uuid and query on collection.properties" do
    collection_response = create_workflow_collection_helper

    # Now create a workflow linked to the collection.
    post "/arvados/v1/workflows",
         params: {:format => :json,
                  :workflow => {
                    collection_uuid: collection_response["uuid"]
                  }
                 },
      headers: auth(:active)
    assert_response :success
    workflow_response = json_response

    # no manifest text by default
    get '/arvados/v1/groups/contents',
        params: {
          filters: [["uuid", "is_a", "arvados#workflow"], ["collection.properties.arv:workflowMain", "=", "foo.cwl"]].to_json,
          include: '["collection_uuid"]',
          format: :json,
        },
        headers: auth(:active)
    assert_response :success
    assert_equal workflow_response["uuid"], json_response["items"][0]["uuid"]
    assert_equal collection_response["uuid"], json_response["included"][0]["uuid"]
    assert_nil json_response["included"][0]["manifest_text"]
    assert_nil json_response["included"][0]["unsigned_manifest_text"]
    assert_equal collection_response["properties"]["arv:workflowMain"], json_response["included"][0]["properties"]["arv:workflowMain"]

    # select didn't include manifest text, so still shouldn't get it
    get '/arvados/v1/groups/contents',
        params: {
          filters: [["uuid", "is_a", "arvados#workflow"], ["collection.properties.arv:workflowMain", "=", "foo.cwl"]].to_json,
          include: '["collection_uuid"]',
          select: '["uuid", "collection_uuid", "properties"]',
          format: :json,
        },
        headers: auth(:active)
    assert_response :success
    assert_equal workflow_response["uuid"], json_response["items"][0]["uuid"]
    assert_equal collection_response["uuid"], json_response["included"][0]["uuid"]
    assert_nil json_response["included"][0]["manifest_text"]
    assert_nil json_response["included"][0]["unsigned_manifest_text"]
    assert_equal collection_response["properties"]["arv:workflowMain"], json_response["included"][0]["properties"]["arv:workflowMain"]

    # currently, with the group contents API, you won't get
    # manifest_text even if you ask for it, because it won't be signed
    # by controller.
    get '/arvados/v1/groups/contents',
        params: {
          filters: [["uuid", "is_a", "arvados#workflow"], ["collection.properties.arv:workflowMain", "=", "foo.cwl"]].to_json,
          include: '["collection_uuid"]',
          select: '["uuid", "collection_uuid", "properties", "manifest_text"]',
          format: :json,
        },
        headers: auth(:active)
    assert_response :success
    assert_equal workflow_response["uuid"], json_response["items"][0]["uuid"]
    assert_equal collection_response["uuid"], json_response["included"][0]["uuid"]
    assert_nil json_response["included"][0]["manifest_text"]
    assert_nil json_response["included"][0]["unsigned_manifest_text"]
    assert_equal collection_response["properties"]["arv:workflowMain"], json_response["included"][0]["properties"]["arv:workflowMain"]

    # However, you can get unsigned_manifest_text
    get '/arvados/v1/groups/contents',
        params: {
          filters: [["uuid", "is_a", "arvados#workflow"], ["collection.properties.arv:workflowMain", "=", "foo.cwl"]].to_json,
          include: '["collection_uuid"]',
          select: '["uuid", "collection_uuid", "properties", "unsigned_manifest_text"]',
          format: :json,
        },
        headers: auth(:active)
    assert_response :success
    assert_equal workflow_response["uuid"], json_response["items"][0]["uuid"]
    assert_equal collection_response["uuid"], json_response["included"][0]["uuid"]
    assert_nil json_response["included"][0]["manifest_text"]
    assert_equal "", json_response["included"][0]["unsigned_manifest_text"]
    assert_equal collection_response["properties"]["arv:workflowMain"], json_response["included"][0]["properties"]["arv:workflowMain"]

  end

end
