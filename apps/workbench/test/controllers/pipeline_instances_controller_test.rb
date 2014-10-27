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


  class RequestDuck
    def self.host
      "localhost"
    end

    def self.port
      8080
    end

    def self.protocol
      "http"
    end
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

    ['component_zzzzz-d1hrv-9fm8l10i9z2kqc9_stage1',
     'component_zzzzz-d1hrv-9fm8l10i9z2kqc9_stage2',
     'zzzzz-8i9sb-graphstage10000',
     'zzzzz-8i9sb-graphstage20000',
     'b519d9cb706a29fc7ea24dbea2f05851+93',
     'fa7aeb5140e2848d39b416daeef4ffc5+45',
     'zzzzz-4zz18-bv31uwvy3neko22',
     'zzzzz-4zz18-uukreo9rbgwsujx'].each do |k|

      assert_not_nil provenance[k], "Expected key #{k} in provenance set"
      assert_equal 1, pips[k], "Expected key #{k} in pips set" if !k.start_with? "component_"
    end

    prov_svg = ProvenanceHelper::create_provenance_graph provenance, "provenance_svg", {
        :request => RequestDuck,
        :all_script_parameters => true,
        :combine_jobs => :script_and_version,
        :pips => pips,
        :only_components => true }

    # hash -> owned_by_active
    assert /hash_4fe459abe02d9b365932b8f5dc419439ab4e2577_99914b932bd37a50b983c5e7c90ae93b&#45;&gt;fa7aeb5140e2848d39b416daeef4ffc5\+45/.match(prov_svg)

    # owned_by_active -> hash2
    assert /fa7aeb5140e2848d39b416daeef4ffc5\+45&#45;&gt;hash2_4fe459abe02d9b365932b8f5dc419439ab4e2577_4900033ec5cfaf8a63566f3664aeaa70/.match(prov_svg)

    #File::open "./tmp/stuff1.svg", "w" do |f|
    #  f.write "<?xml version=\"1.0\" ?>\n"
    #  f.write prov_svg
    #end

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

    [['component_zzzzz-d1hrv-9fm8l10i9z2kqc9_stage1', nil],
     ['component_zzzzz-d1hrv-9fm8l10i9z2kqc9_stage2', nil],
     ['component_zzzzz-d1hrv-9fm8l10i9z2kqc0_stage1', nil],
     ['component_zzzzz-d1hrv-9fm8l10i9z2kqc0_stage2', nil],
     ['zzzzz-8i9sb-graphstage10000', 3],
     ['zzzzz-8i9sb-graphstage20000', 1],
     ['zzzzz-8i9sb-graphstage30000', 2],
     ['b519d9cb706a29fc7ea24dbea2f05851+93', 1],
     ['fa7aeb5140e2848d39b416daeef4ffc5+45', 3],
     ['ea10d51bcf88862dbcc36eb292017dfd+45', 2],
     ['zzzzz-4zz18-bv31uwvy3neko22', 3],
     ['zzzzz-4zz18-uukreo9rbgwsujx', 1],
     ['zzzzz-4zz18-uukreo9rbgwsujj', 2]
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

    # owned_by_active -> hash2 (stuff)
    assert /fa7aeb5140e2848d39b416daeef4ffc5\+45&#45;&gt;hash2_4fe459abe02d9b365932b8f5dc419439ab4e2577_4900033ec5cfaf8a63566f3664aeaa70/.match(prov_svg)

    # owned_by_active -> hash2 (stuff2)
    assert /fa7aeb5140e2848d39b416daeef4ffc5\+45&#45;&gt;hash2_4fe459abe02d9b365932b8f5dc419439ab4e2577_02a085407e751d00b5dc88f1bd5e8247/.match(prov_svg)

    # hash2 (stuff) -> GPL
    assert /hash2_4fe459abe02d9b365932b8f5dc419439ab4e2577_4900033ec5cfaf8a63566f3664aeaa70&#45;&gt;b519d9cb706a29fc7ea24dbea2f05851\+93/.match(prov_svg)

    # hash2 (stuff2) -> baz file
    assert /hash2_4fe459abe02d9b365932b8f5dc419439ab4e2577_02a085407e751d00b5dc88f1bd5e8247&#45;&gt;ea10d51bcf88862dbcc36eb292017dfd\+45/.match(prov_svg)

    # File::open "./tmp/stuff2.svg", "w" do |f|
    #   f.write "<?xml version=\"1.0\" ?>\n"
    #   f.write prov_svg
    # end

  end

end
