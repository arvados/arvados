require 'test_helper'

class CollectionTest < ActiveSupport::TestCase
  def create_collection name, enc=nil
    txt = ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:#{name}.txt\n"
    txt.force_encoding(enc) if enc
    return Collection.create(manifest_text: txt)
  end

  test 'accept ASCII manifest_text' do
    act_as_system_user do
      c = create_collection 'foo', Encoding::US_ASCII
      assert c.valid?
    end
  end

  test 'accept UTF-8 manifest_text' do
    act_as_system_user do
      c = create_collection "f\xc3\x98\xc3\x98", Encoding::UTF_8
      assert c.valid?
    end
  end

  test 'refuse manifest_text with invalid UTF-8 byte sequence' do
    act_as_system_user do
      c = create_collection "f\xc8o", Encoding::UTF_8
      assert !c.valid?
      assert_equal [:manifest_text], c.errors.messages.keys
      assert_match /UTF-8/, c.errors.messages[:manifest_text].first
    end
  end

  test 'refuse manifest_text with non-UTF-8 encoding' do
    act_as_system_user do
      c = create_collection "f\xc8o", Encoding::ASCII_8BIT
      assert !c.valid?
      assert_equal [:manifest_text], c.errors.messages.keys
      assert_match /UTF-8/, c.errors.messages[:manifest_text].first
    end
  end

  test 'create and update collection and verify file_names' do
    act_as_system_user do
      c = create_collection 'foo', Encoding::US_ASCII
      assert c.valid?
      created_file_names = c.file_names
      assert created_file_names
      assert_match /foo.txt/, c.file_names

      c.update_attribute 'manifest_text', ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo2.txt\n"
      assert_not_equal created_file_names, c.file_names
      assert_match /foo2.txt/, c.file_names
    end
  end

  [
    [2**8, false],
    [2**18, true],
  ].each do |manifest_size, gets_truncated|
    test "create collection with manifest size #{manifest_size} which gets truncated #{gets_truncated},
          and not expect exceptions even on very large manifest texts" do
      # file_names has a max size, hence there will be no errors even on large manifests
      act_as_system_user do
        manifest_text = './blurfl d41d8cd98f00b204e9800998ecf8427e+0'
        index = 0
        while manifest_text.length < manifest_size
          manifest_text += ' ' + "0:0:veryverylongfilename000000000000#{index}.txt\n./subdir1"
          index += 1
        end
        manifest_text += "\n"
        c = Collection.create(manifest_text: manifest_text)

        assert c.valid?
        assert c.file_names
        assert_match /veryverylongfilename0000000000001.txt/, c.file_names
        assert_match /veryverylongfilename0000000000002.txt/, c.file_names
        if !gets_truncated
          assert_match /blurfl/, c.file_names
          assert_match /subdir1/, c.file_names
        end
      end
    end
  end

  [
    ['foo', true],
    ['foo bar', true],
    ['foox barx', false],                               # no match for both
    ['foox bar', true],                                 # bar matches
    ['foo barx', true],
    ['file2_in_subdir4', true],                         # whole string match
    ['filex_in_subdir4', false],                        # looks for the whole string and fails
    ['filex in subdir4', true],                         # matches subdir4
    ['6a4ff0499484c6c79c95cd8c566bd25f+249025', true],
    ['6a4ff0499484c6c79c95cd8c566bd25f+249024', false], # matches the whole string and fails
    ['6a4ff0499484c6c79c95cd8', true],                  # prefix matches    
    ['499484c6c79c95cd8c566bd', false],                 # not a prefix match
    ['no-such-file', false],                            # looks for whole string and fails
    ['no such file', true],                             # matches "file"
  ].each do |search_filter, expect_results|
    test "full text search collection for #{search_filter} and expect results #{expect_results}" do
      search_filters = search_filter.split.each {|s| s.concat(':*')}
      results = Collection.where("to_tsvector('english', translate(manifest_text, '/.', '  ')) @@ to_tsquery(?)",
                                 "#{search_filters.join('|')}")
      if expect_results
        assert_equal true, results.length>0
      else
        assert_equal 0, results.length
      end
    end
  end
end
