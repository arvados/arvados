require 'test_helper'

class ActionsControllerTest < ActionController::TestCase

  test "send report" do
    post :report_issue, {format: 'js'}, session_for(:admin)
    assert_response :success

    found_email = false
    ActionMailer::Base.deliveries.andand.each do |email|
      if email.subject.include? "Issue reported by admin"
        found_email = true
        break
      end
    end
    assert_equal true, found_email, 'Expected email after issue reported'
  end

  test "combine files into new collection" do
    post(:combine_selected_files_into_collection, {
           selection: ['zzzzz-4zz18-znfnqtbbv4spc3w/foo',
                       'zzzzz-4zz18-ehbhgtheo8909or/bar',
                       'zzzzz-4zz18-y9vne9npefyxh8g/baz',
                       '7a6ef4c162a5c6413070a8bd0bffc818+150'],
           format: "json"},
         session_for(:active))

    assert_response 302   # collection created and redirected to new collection page

    assert response.headers['Location'].include? '/collections/'
    new_collection_uuid = response.headers['Location'].split('/')[-1]

    use_token :active
    collection = Collection.select([:uuid, :manifest_text]).where(uuid: new_collection_uuid).first
    manifest_text = collection['manifest_text']
    assert manifest_text.include?('foo'), 'Not found foo in new collection manifest text'
    assert manifest_text.include?('bar'), 'Not found bar in new collection manifest text'
    assert manifest_text.include?('baz'), 'Not found baz in new collection manifest text'
    assert manifest_text.include?('0:0:file1 0:0:file2 0:0:file3'),
                'Not found 0:0:file1 0:0:file2 0:0:file3 in new collection manifest text'
    assert manifest_text.include?('dir1/subdir'), 'Not found dir1/subdir in new collection manifest text'
    assert manifest_text.include?('dir2'), 'Not found dir2 in new collection manifest text'
  end

  test "combine files  with repeated names into new collection" do
    post(:combine_selected_files_into_collection, {
           selection: ['zzzzz-4zz18-znfnqtbbv4spc3w/foo',
                       'zzzzz-4zz18-00000nonamecoll/foo',
                       'zzzzz-4zz18-abcd6fx123409f7/foo',
                       'zzzzz-4zz18-ehbhgtheo8909or/bar',
                       'zzzzz-4zz18-y9vne9npefyxh8g/baz',
                       '7a6ef4c162a5c6413070a8bd0bffc818+150'],
           format: "json"},
         session_for(:active))

    assert_response 302   # collection created and redirected to new collection page

    assert response.headers['Location'].include? '/collections/'
    new_collection_uuid = response.headers['Location'].split('/')[-1]

    use_token :active
    collection = Collection.select([:uuid, :manifest_text]).where(uuid: new_collection_uuid).first
    manifest_text = collection['manifest_text']
    assert manifest_text.include?('foo'), 'Not found foo in new collection manifest text'
    assert manifest_text.include?('foo(1)'), 'Not found foo(1) in new collection manifest text'
    assert manifest_text.include?('foo(2)'), 'Not found foo(2) in new collection manifest text'
    assert manifest_text.include?('bar'), 'Not found bar in new collection manifest text'
    assert manifest_text.include?('baz'), 'Not found baz in new collection manifest text'
    assert manifest_text.include?('0:0:file1 0:0:file2 0:0:file3'),
                'Not found 0:0:file1 0:0:file2 0:0:file3 in new collection manifest text'
    assert manifest_text.include?('dir1/subdir'), 'Not found dir1/subdir in new collection manifest text'
    assert manifest_text.include?('dir2'), 'Not found dir2 in new collection manifest text'
  end

  test "combine collections with repeated filenames in almost similar directories and expect files with proper suffixes" do
    post(:combine_selected_files_into_collection, {
           selection: ['zzzzz-4zz18-duplicatenames1',
                       'zzzzz-4zz18-duplicatenames2',
                       'zzzzz-4zz18-znfnqtbbv4spc3w/foo',
                       'zzzzz-4zz18-00000nonamecoll/foo',],
           format: "json"},
         session_for(:active))

    assert_response 302   # collection created and redirected to new collection page

    assert response.headers['Location'].include? '/collections/'
    new_collection_uuid = response.headers['Location'].split('/')[-1]

    use_token :active
    collection = Collection.select([:uuid, :manifest_text]).where(uuid: new_collection_uuid).first
    manifest_text = collection['manifest_text']

    assert manifest_text.include?('foo'), 'Not found foo in new collection manifest text'
    assert manifest_text.include?('foo(1)'), 'Not found foo(1) in new collection manifest text'

    streams = manifest_text.split "\n"
    streams.each do |stream|
      if stream.start_with? './dir1'
        # dir1 stream
        assert stream.include?(':alice(1)'), 'Not found: alice(1) in dir1'
        assert stream.include?(':alice.txt'), 'Not found: alice.txt in dir1'
        assert stream.include?(':alice(1).txt'), 'Not found: alice(1).txt in dir1'
        assert stream.include?(':bob.txt'), 'Not found: bob.txt in dir1'
        assert stream.include?(':carol.txt'), 'Not found: carol.txt in dir1'
      elsif stream.start_with? './dir2'
        # dir2 stream
        assert stream.include?(':alice.txt'), 'Not found: alice.txt in dir2'
        assert stream.include?(':alice(1).txt'), 'Not found: alice(1).txt in dir2'
      elsif stream.start_with? '. '
        # . stream
        assert stream.include?(':foo'), 'Not found: foo in .'
        assert stream.include?(':foo(1)'), 'Not found: foo(1) in .'
      end
    end
  end

  test "combine collections with same filename in two different streams and expect no suffixes for filenames" do
    post(:combine_selected_files_into_collection, {
           selection: ['zzzzz-4zz18-znfnqtbbv4spc3w',
                       'zzzzz-4zz18-foonbarfilesdir'],
           format: "json"},
         session_for(:active))

    assert_response 302   # collection created and redirected to new collection page

    assert response.headers['Location'].include? '/collections/'
    new_collection_uuid = response.headers['Location'].split('/')[-1]

    use_token :active
    collection = Collection.select([:uuid, :manifest_text]).where(uuid: new_collection_uuid).first
    manifest_text = collection['manifest_text']

    streams = manifest_text.split "\n"
    assert_equal 2, streams.length
    streams.each do |stream|
      if stream.start_with? './dir1'
        assert stream.include?('foo'), 'Not found: foo in dir1'
      elsif stream.start_with? '. '
        assert stream.include?('foo'), 'Not found: foo in .'
      end
    end
    assert !manifest_text.include?('foo(1)'), 'Found foo(1) in new collection manifest text'
  end

  test "combine foo files from two different collection streams and expect proper filename suffixes" do
    post(:combine_selected_files_into_collection, {
           selection: ['zzzzz-4zz18-znfnqtbbv4spc3w/foo',
                       'zzzzz-4zz18-foonbarfilesdir/dir1/foo'],
           format: "json"},
         session_for(:active))

    assert_response 302   # collection created and redirected to new collection page

    assert response.headers['Location'].include? '/collections/'
    new_collection_uuid = response.headers['Location'].split('/')[-1]

    use_token :active
    collection = Collection.select([:uuid, :manifest_text]).where(uuid: new_collection_uuid).first
    manifest_text = collection['manifest_text']

    streams = manifest_text.split "\n"
    assert_equal 1, streams.length
    assert manifest_text.include?('foo'), 'Found foo in new collection manifest text'
    assert manifest_text.include?('foo(1)'), 'Found foo(1) in new collection manifest text'
  end
end
