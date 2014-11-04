require 'test_helper'

class PipelineInstancesControllerTest < ActionController::TestCase
  include PipelineInstancesHelper

  test "one" do
    r = [{started_at: 1, finished_at: 3}]
    assert_equal 2, determine_wallclock_runtime(r)

    r = [{started_at: 1, finished_at: 5}]
    assert_equal 4, determine_wallclock_runtime(r)

    r = [{started_at: 1, finished_at: 2}, {started_at: 3, finished_at: 5}]
    assert_equal 3, determine_wallclock_runtime(r)

    r = [{started_at: 3, finished_at: 5}, {started_at: 1, finished_at: 2}]
    assert_equal 3, determine_wallclock_runtime(r)

    r = [{started_at: 3, finished_at: 5}, {started_at: 1, finished_at: 2},
         {started_at: 2, finished_at: 4}]
    assert_equal 4, determine_wallclock_runtime(r)

    r = [{started_at: 1, finished_at: 5}, {started_at: 2, finished_at: 3}]
    assert_equal 4, determine_wallclock_runtime(r)

    r = [{started_at: 3, finished_at: 5}, {started_at: 1, finished_at: 4}]
    assert_equal 4, determine_wallclock_runtime(r)

    r = [{started_at: 1, finished_at: 4}, {started_at: 3, finished_at: 5}]
    assert_equal 4, determine_wallclock_runtime(r)

    r = [{started_at: 1, finished_at: 4}, {started_at: 3, finished_at: 5},
         {started_at: 5, finished_at: 8}]
    assert_equal 7, determine_wallclock_runtime(r)

    r = [{started_at: 1, finished_at: 4}, {started_at: 3, finished_at: 5},
         {started_at: 6, finished_at: 8}]
    assert_equal 6, determine_wallclock_runtime(r)
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
