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

    assert_includes(response.headers['Location'], '/collections/')
    new_collection_uuid = response.headers['Location'].split('/')[-1]

    use_token :active
    collection = Collection.select([:uuid, :manifest_text]).where(uuid: new_collection_uuid).first
    manifest_text = collection['manifest_text']
    assert_includes(manifest_text, "foo")
    assert_includes(manifest_text, "bar")
    assert_includes(manifest_text, "baz")
    assert_includes(manifest_text, "0:0:file1 0:0:file2 0:0:file3")
    assert_includes(manifest_text, "dir1/subdir")
    assert_includes(manifest_text, "dir2")
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

    assert_includes(response.headers['Location'], '/collections/')
    new_collection_uuid = response.headers['Location'].split('/')[-1]

    use_token :active
    collection = Collection.select([:uuid, :manifest_text]).where(uuid: new_collection_uuid).first
    manifest_text = collection['manifest_text']
    assert_includes(manifest_text, "foo(1)")
    assert_includes(manifest_text, "foo(2)")
    assert_includes(manifest_text, "bar")
    assert_includes(manifest_text, "baz")
    assert_includes(manifest_text, "0:0:file1 0:0:file2 0:0:file3")
    assert_includes(manifest_text, "dir1/subdir")
    assert_includes(manifest_text, "dir2")
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

    assert_includes(manifest_text, 'foo')
    assert_includes(manifest_text, 'foo(1)')

    streams = manifest_text.split "\n"
    streams.each do |stream|
      if stream.start_with? './dir1'
        # dir1 stream
        assert_includes(stream, ':alice(1)')
        assert_includes(stream, ':alice.txt')
        assert_includes(stream, ':alice(1).txt')
        assert_includes(stream, ':bob.txt')
        assert_includes(stream, ':carol.txt')
      elsif stream.start_with? './dir2'
        # dir2 stream
        assert_includes(stream, ':alice.txt')
        assert_includes(stream, ':alice(1).txt')
      elsif stream.start_with? '. '
        # . stream
        assert_includes(stream, ':foo')
        assert_includes(stream, ':foo(1)')
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

    assert_includes(response.headers['Location'], '/collections/')
    new_collection_uuid = response.headers['Location'].split('/')[-1]

    use_token :active
    collection = Collection.select([:uuid, :manifest_text]).where(uuid: new_collection_uuid).first
    manifest_text = collection['manifest_text']

    streams = manifest_text.split "\n"
    assert_equal 2, streams.length
    streams.each do |stream|
      if stream.start_with? './dir1'
        assert_includes(stream, 'foo')
      elsif stream.start_with? '. '
        assert_includes(stream, 'foo')
      end
    end
    refute_includes(manifest_text, 'foo(1)')
  end

  test "combine foo files from two different collection streams and expect proper filename suffixes" do
    post(:combine_selected_files_into_collection, {
           selection: ['zzzzz-4zz18-znfnqtbbv4spc3w/foo',
                       'zzzzz-4zz18-foonbarfilesdir/dir1/foo'],
           format: "json"},
         session_for(:active))

    assert_response 302   # collection created and redirected to new collection page

    assert_includes(response.headers['Location'], '/collections/')
    new_collection_uuid = response.headers['Location'].split('/')[-1]

    use_token :active
    collection = Collection.select([:uuid, :manifest_text]).where(uuid: new_collection_uuid).first
    manifest_text = collection['manifest_text']

    streams = manifest_text.split "\n"
    assert_equal 1, streams.length, "Incorrect number of streams in #{manifest_text}"
    assert_includes(manifest_text, 'foo')
    assert_includes(manifest_text, 'foo(1)')
  end

  [
    ['collections', 'user_agreement_in_anonymously_accessible_project'],
    ['groups', 'anonymously_accessible_project'],
    ['jobs', 'running_job_in_publicly_accessible_project'],
    ['pipeline_instances', 'pipeline_in_publicly_accessible_project'],
    ['pipeline_templates', 'pipeline_template_in_publicly_accessible_project'],
  ].each do |dm, fixture|
    test "access show method for public #{dm} and expect to see page" do
      Rails.configuration.anonymous_user_token = api_fixture('api_client_authorizations')['anonymous']['api_token']
      get(:show, {uuid: api_fixture(dm)[fixture]['uuid']})
      assert_response :redirect
      if dm == 'groups'
        assert_includes @response.redirect_url, "projects/#{fixture['uuid']}"
      else
        assert_includes @response.redirect_url, "#{dm}/#{fixture['uuid']}"
      end
    end
  end

  [
    ['collections', 'foo_collection_in_aproject', 404],
    ['groups', 'subproject_in_asubproject_with_same_name_as_one_in_active_user_home', 404],
    ['jobs', 'job_with_latest_version', 404],
    ['pipeline_instances', 'pipeline_owned_by_active_in_home', 404],
    ['pipeline_templates', 'template_in_asubproject_with_same_name_as_one_in_active_user_home', 404],
    ['traits', 'owned_by_aproject_with_no_name', :redirect],
  ].each do |dm, fixture, expected|
    test "access show method for non-public #{dm} and expect #{expected}" do
      Rails.configuration.anonymous_user_token = api_fixture('api_client_authorizations')['anonymous']['api_token']
      get(:show, {uuid: api_fixture(dm)[fixture]['uuid']})
      assert_response expected
      if expected == 404
        assert_includes @response.inspect, 'Log in'
      else
        assert_match /\/users\/welcome/, @response.redirect_url
      end
    end
  end
end
