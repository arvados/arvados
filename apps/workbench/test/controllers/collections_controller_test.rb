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
    obj = Collection.where(uuid: 'zzzzz-4zz18-uukreo9rbgwsujj').results.first

    provenance = obj.provenance.stringify_keys

    [obj[:portable_data_hash]].each do |k|
      assert_not_nil provenance[k], "Expected key #{k} in provenance set"
    end

    prov_svg = ProvenanceHelper::create_provenance_graph(provenance, "provenance_svg",
                                                         {:request => RequestDuck,
                                                           :direction => :bottom_up,
                                                           :combine_jobs => :script_only})

    # hash -> baz file
    assert /ea10d51bcf88862dbcc36eb292017dfd\+45&#45;&gt;hash_f866587e2de5291fbd38d616d6d33eab/.match(prov_svg)

    # hash2 -> baz file
    assert /ea10d51bcf88862dbcc36eb292017dfd\+45&#45;&gt;hash2_02a085407e751d00b5dc88f1bd5e8247/.match(prov_svg)

    # owned_by_active -> hash
    assert /hash_f866587e2de5291fbd38d616d6d33eab&#45;&gt;fa7aeb5140e2848d39b416daeef4ffc5\+45/.match(prov_svg)

    # owned_by_active -> hash2
    assert /hash2_02a085407e751d00b5dc88f1bd5e8247&#45;&gt;fa7aeb5140e2848d39b416daeef4ffc5\+45/.match(prov_svg)

    # File::open "./tmp/stuff3.svg", "w" do |f|
    #   f.write "<?xml version=\"1.0\" ?>\n"
    #   f.write prov_svg
    # end

  end

  test 'used_by graph' do
    use_token 'admin'
    obj = Collection.where(uuid: 'zzzzz-4zz18-bv31uwvy3neko22').results.first

    used_by = obj.used_by.stringify_keys

    used_by_svg = ProvenanceHelper::create_provenance_graph(used_by, "used_by_svg",
                                                            {:request => RequestDuck,
                                                              :direction => :top_down,
                                                              :combine_jobs => :script_only,
                                                              :pdata_only => true})

    # bar_file -> hash2
    assert /fa7aeb5140e2848d39b416daeef4ffc5\+45&#45;&gt;hash2_f866587e2de5291fbd38d616d6d33eab/.match(used_by_svg)

    # hash -> baz file
    assert /hash_f866587e2de5291fbd38d616d6d33eab&#45;&gt;ea10d51bcf88862dbcc36eb292017dfd\+45/.match(used_by_svg)

    # hash2 -> baz file
    assert /hash2_02a085407e751d00b5dc88f1bd5e8247&#45;&gt;ea10d51bcf88862dbcc36eb292017dfd\+45/.match(used_by_svg)


    # File::open "./tmp/stuff4.svg", "w" do |f|
    #   f.write "<?xml version=\"1.0\" ?>\n"
    #   f.write used_by_svg
    # end

  end
end
