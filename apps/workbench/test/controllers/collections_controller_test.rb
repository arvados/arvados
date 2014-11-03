require 'test_helper'

class CollectionsControllerTest < ActionController::TestCase
  include PipelineInstancesHelper

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


  test 'provenance graph' do
    use_token 'admin'

    obj = find_fixture Collection, "graph_test_collection3"

    provenance = obj.provenance.stringify_keys

    [obj[:portable_data_hash]].each do |k|
      assert_not_nil provenance[k], "Expected key #{k} in provenance set"
    end

    prov_svg = ProvenanceHelper::create_provenance_graph(provenance, "provenance_svg",
                                                         {:request => RequestDuck,
                                                           :direction => :bottom_up,
                                                           :combine_jobs => :script_only})

    stage1 = find_fixture Job, "graph_stage1"
    stage3 = find_fixture Job, "graph_stage3"
    previous_job_run = find_fixture Job, "previous_job_run"

    obj_id = obj.portable_data_hash.gsub('+', '\\\+')
    stage1_out = stage1.output.gsub('+', '\\\+')
    stage1_id = "#{stage1.script}_#{Digest::MD5.hexdigest(stage1[:script_parameters].to_json)}"
    stage3_id = "#{stage3.script}_#{Digest::MD5.hexdigest(stage3[:script_parameters].to_json)}"

    assert /#{obj_id}&#45;&gt;#{stage3_id}/.match(prov_svg)

    assert /#{stage3_id}&#45;&gt;#{stage1_out}/.match(prov_svg)

    assert /#{stage1_out}&#45;&gt;#{stage1_id}/.match(prov_svg)

  end

  test 'used_by graph' do
    use_token 'admin'
    obj = find_fixture Collection, "graph_test_collection1"

    used_by = obj.used_by.stringify_keys

    used_by_svg = ProvenanceHelper::create_provenance_graph(used_by, "used_by_svg",
                                                            {:request => RequestDuck,
                                                              :direction => :top_down,
                                                              :combine_jobs => :script_only,
                                                              :pdata_only => true})

    stage2 = find_fixture Job, "graph_stage2"
    stage3 = find_fixture Job, "graph_stage3"

    stage2_id = "#{stage2.script}_#{Digest::MD5.hexdigest(stage2[:script_parameters].to_json)}"
    stage3_id = "#{stage3.script}_#{Digest::MD5.hexdigest(stage3[:script_parameters].to_json)}"

    obj_id = obj.portable_data_hash.gsub('+', '\\\+')
    stage3_out = stage3.output.gsub('+', '\\\+')

    assert /#{obj_id}&#45;&gt;#{stage2_id}/.match(used_by_svg)

    assert /#{obj_id}&#45;&gt;#{stage3_id}/.match(used_by_svg)

    assert /#{stage3_id}&#45;&gt;#{stage3_out}/.match(used_by_svg)

    assert /#{stage3_id}&#45;&gt;#{stage3_out}/.match(used_by_svg)

  end
end
