require 'test_helper'

class PipelineInstancesControllerTest < ActionController::TestCase
  include PipelineInstancesHelper

  def create_instance_long_enough_to(instance_attrs={})
    # create 'two_part' pipeline with the given instance attributes
    pt_fixture = api_fixture('pipeline_templates')['two_part']
    post :create, {
      pipeline_instance: instance_attrs.merge({
        pipeline_template_uuid: pt_fixture['uuid']
      }),
      format: :json
    }, session_for(:active)
    assert_response :success
    pi_uuid = assigns(:object).uuid
    assert_not_nil assigns(:object)

    # yield
    yield pi_uuid, pt_fixture

    # delete the pipeline instance
    use_token :active
    PipelineInstance.where(uuid: pi_uuid).first.destroy
  end

  test "pipeline instance components populated after create" do
    create_instance_long_enough_to do |new_instance_uuid, template_fixture|
      assert_equal(template_fixture['components'].to_json,
                   assigns(:object).components.to_json)
    end
  end

  test "can render pipeline instance with tagged collections" do
    # Make sure to pass in a tagged collection to test that part of the rendering behavior.
    get(:show,
        {id: api_fixture("pipeline_instances")["pipeline_with_tagged_collection_input"]["uuid"]},
        session_for(:active))
    assert_response :success
  end

  test "update script_parameters one at a time using merge param" do
      template_fixture = api_fixture('pipeline_templates')['two_part']
      post :update, {
        id: api_fixture("pipeline_instances")["pipeline_to_merge_params"]["uuid"],
        pipeline_instance: {
          components: {
            "part-two" => {
              script_parameters: {
                integer_with_value: {
                  value: 9
                },
                plain_string: {
                  value: 'quux'
                },
              }
            }
          }
        },
        merge: true,
        format: :json
      }, session_for(:active)
      assert_response :success
      assert_not_nil assigns(:object)
      orig_params = template_fixture['components']['part-two']['script_parameters']
      new_params = assigns(:object).components[:'part-two'][:script_parameters]
      orig_params.keys.each do |k|
        unless %w(integer_with_value plain_string).index(k)
          assert_equal orig_params[k].to_json, new_params[k.to_sym].to_json
        end
      end
  end

  test "component rendering copes with unexpected components format" do
    get(:show,
        {id: api_fixture("pipeline_instances")["components_is_jobspec"]["uuid"]},
        session_for(:active))
    assert_response :success
  end

  test "dates in JSON components are parsed" do
    get(:show,
        {id: api_fixture('pipeline_instances')['has_component_with_completed_jobs']['uuid']},
        session_for(:active))
    assert_response :success
    assert_not_nil assigns(:object)
    assert_not_nil assigns(:object).components[:foo][:job]
    assert assigns(:object).components[:foo][:job][:started_at].is_a? Time
    assert assigns(:object).components[:foo][:job][:finished_at].is_a? Time
  end

  # The next two tests ensure that a pipeline instance can be copied
  # when the template has components that do not exist in the
  # instance (ticket #4000).

  test "copy pipeline instance with components=use_latest" do
    post(:copy,
         {
           id: api_fixture('pipeline_instances')['pipeline_with_newer_template']['uuid'],
           components: 'use_latest',
           script: 'use_latest',
           pipeline_instance: {
             state: 'RunningOnServer'
           }
         },
         session_for(:active))
    assert_response 302
    assert_not_nil assigns(:object)

    # Component 'foo' has script parameters only in the pipeline instance.
    # Component 'bar' is present only in the pipeline_template.
    # Test that the copied pipeline instance includes parameters for
    # component 'foo' from the source instance, and parameters for
    # component 'bar' from the source template.
    #
    assert_not_nil assigns(:object).components[:foo]
    foo = assigns(:object).components[:foo]
    assert_not_nil foo[:script_parameters]
    assert_not_nil foo[:script_parameters][:input]
    assert_equal 'foo instance input', foo[:script_parameters][:input][:title]

    assert_not_nil assigns(:object).components[:bar]
    bar = assigns(:object).components[:bar]
    assert_not_nil bar[:script_parameters]
    assert_not_nil bar[:script_parameters][:input]
    assert_equal 'bar template input', bar[:script_parameters][:input][:title]
  end

  test "copy pipeline instance on newer template works with script=use_same" do
    post(:copy,
         {
           id: api_fixture('pipeline_instances')['pipeline_with_newer_template']['uuid'],
           components: 'use_latest',
           script: 'use_same',
           pipeline_instance: {
             state: 'RunningOnServer'
           }
         },
         session_for(:active))
    assert_response 302
    assert_not_nil assigns(:object)

    # Test that relevant component parameters were copied from both
    # the source instance and source template, respectively (see
    # previous test)
    #
    assert_not_nil assigns(:object).components[:foo]
    foo = assigns(:object).components[:foo]
    assert_not_nil foo[:script_parameters]
    assert_not_nil foo[:script_parameters][:input]
    assert_equal 'foo instance input', foo[:script_parameters][:input][:title]

    assert_not_nil assigns(:object).components[:bar]
    bar = assigns(:object).components[:bar]
    assert_not_nil bar[:script_parameters]
    assert_not_nil bar[:script_parameters][:input]
    assert_equal 'bar template input', bar[:script_parameters][:input][:title]
  end

  test "generate graph" do

    use_token 'admin'

    pipeline_for_graph = {
      state: 'Complete',
      uuid: 'zzzzz-d1hrv-9fm8l10i9z2kqc9',
      components: {
        stage1: {
          repository: 'foo',
          script: 'hash',
          script_version: 'master',
          job: {uuid: 'zzzzz-8i9sb-graphstage10000'},
          output_uuid: 'zzzzz-4zz18-bv31uwvy3neko22'
        },
        stage2: {
          repository: 'foo',
          script: 'hash2',
          script_version: 'master',
          script_parameters: {
            input: 'fa7aeb5140e2848d39b416daeef4ffc5+45'
          },
          job: {uuid: 'zzzzz-8i9sb-graphstage20000'},
          output_uuid: 'zzzzz-4zz18-uukreo9rbgwsujx'
        }
      }
    }

    @controller.params['tab_pane'] = "Graph"
    provenance, pips = @controller.graph([pipeline_for_graph])

    graph_test_collection1 = find_fixture Collection, "graph_test_collection1"
    stage1 = find_fixture Job, "graph_stage1"
    stage2 = find_fixture Job, "graph_stage2"

    ['component_zzzzz-d1hrv-9fm8l10i9z2kqc9_stage1',
     'component_zzzzz-d1hrv-9fm8l10i9z2kqc9_stage2',
     stage1.uuid,
     stage2.uuid,
     stage1.output,
     stage2.output,
     pipeline_for_graph[:components][:stage1][:output_uuid],
     pipeline_for_graph[:components][:stage2][:output_uuid]
    ].each do |k|

      assert_not_nil provenance[k], "Expected key #{k} in provenance set"
      assert_equal 1, pips[k], "Expected key #{k} in pips set" if !k.start_with? "component_"
    end

    prov_svg = ProvenanceHelper::create_provenance_graph provenance, "provenance_svg", {
        :request => RequestDuck,
        :all_script_parameters => true,
        :combine_jobs => :script_and_version,
        :pips => pips,
        :only_components => true }

    stage1_id = "#{stage1[:script]}_#{stage1[:script_version]}_#{Digest::MD5.hexdigest(stage1[:script_parameters].to_json)}"
    stage2_id = "#{stage2[:script]}_#{stage2[:script_version]}_#{Digest::MD5.hexdigest(stage2[:script_parameters].to_json)}"

    stage1_out = stage1[:output].gsub('+','\\\+')

    assert_match /#{stage1_id}&#45;&gt;#{stage1_out}/, prov_svg

    assert_match /#{stage1_out}&#45;&gt;#{stage2_id}/, prov_svg

  end

  test "generate graph compare" do

    use_token 'admin'

    pipeline_for_graph1 = {
      state: 'Complete',
      uuid: 'zzzzz-d1hrv-9fm8l10i9z2kqc9',
      components: {
        stage1: {
          repository: 'foo',
          script: 'hash',
          script_version: 'master',
          job: {uuid: 'zzzzz-8i9sb-graphstage10000'},
          output_uuid: 'zzzzz-4zz18-bv31uwvy3neko22'
        },
        stage2: {
          repository: 'foo',
          script: 'hash2',
          script_version: 'master',
          script_parameters: {
            input: 'fa7aeb5140e2848d39b416daeef4ffc5+45'
          },
          job: {uuid: 'zzzzz-8i9sb-graphstage20000'},
          output_uuid: 'zzzzz-4zz18-uukreo9rbgwsujx'
        }
      }
    }

    pipeline_for_graph2 = {
      state: 'Complete',
      uuid: 'zzzzz-d1hrv-9fm8l10i9z2kqc0',
      components: {
        stage1: {
          repository: 'foo',
          script: 'hash',
          script_version: 'master',
          job: {uuid: 'zzzzz-8i9sb-graphstage10000'},
          output_uuid: 'zzzzz-4zz18-bv31uwvy3neko22'
        },
        stage2: {
          repository: 'foo',
          script: 'hash2',
          script_version: 'master',
          script_parameters: {
          },
          job: {uuid: 'zzzzz-8i9sb-graphstage30000'},
          output_uuid: 'zzzzz-4zz18-uukreo9rbgwsujj'
        }
      }
    }

    @controller.params['tab_pane'] = "Graph"
    provenance, pips = @controller.graph([pipeline_for_graph1, pipeline_for_graph2])

    collection1 = find_fixture Collection, "graph_test_collection1"

    stage1 = find_fixture Job, "graph_stage1"
    stage2 = find_fixture Job, "graph_stage2"
    stage3 = find_fixture Job, "graph_stage3"

    [['component_zzzzz-d1hrv-9fm8l10i9z2kqc9_stage1', nil],
     ['component_zzzzz-d1hrv-9fm8l10i9z2kqc9_stage2', nil],
     ['component_zzzzz-d1hrv-9fm8l10i9z2kqc0_stage1', nil],
     ['component_zzzzz-d1hrv-9fm8l10i9z2kqc0_stage2', nil],
     [stage1.uuid, 3],
     [stage2.uuid, 1],
     [stage3.uuid, 2],
     [stage1.output, 3],
     [stage2.output, 1],
     [stage3.output, 2],
     [pipeline_for_graph1[:components][:stage1][:output_uuid], 3],
     [pipeline_for_graph1[:components][:stage2][:output_uuid], 1],
     [pipeline_for_graph2[:components][:stage2][:output_uuid], 2]
    ].each do |k|
      assert_not_nil provenance[k[0]], "Expected key #{k[0]} in provenance set"
      assert_equal k[1], pips[k[0]], "Expected key #{k} in pips" if !k[0].start_with? "component_"
    end

    prov_svg = ProvenanceHelper::create_provenance_graph provenance, "provenance_svg", {
        :request => RequestDuck,
        :all_script_parameters => true,
        :combine_jobs => :script_and_version,
        :pips => pips,
        :only_components => true }

    collection1_id = collection1.portable_data_hash.gsub('+','\\\+')

    stage2_id = "#{stage2[:script]}_#{stage2[:script_version]}_#{Digest::MD5.hexdigest(stage2[:script_parameters].to_json)}"
    stage3_id = "#{stage3[:script]}_#{stage3[:script_version]}_#{Digest::MD5.hexdigest(stage3[:script_parameters].to_json)}"

    stage2_out = stage2[:output].gsub('+','\\\+')
    stage3_out = stage3[:output].gsub('+','\\\+')

    assert_match /#{collection1_id}&#45;&gt;#{stage2_id}/, prov_svg
    assert_match /#{collection1_id}&#45;&gt;#{stage3_id}/, prov_svg

    assert_match /#{stage2_id}&#45;&gt;#{stage2_out}/, prov_svg
    assert_match /#{stage3_id}&#45;&gt;#{stage3_out}/, prov_svg

  end

end
