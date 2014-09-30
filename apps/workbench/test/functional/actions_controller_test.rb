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
                       '1fd08fc162a5c6413070a8bd0bffc818+150'],
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

end
