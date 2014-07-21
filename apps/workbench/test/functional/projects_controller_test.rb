require 'test_helper'

class ProjectsControllerTest < ActionController::TestCase
  test "inactive user is asked to sign user agreements on front page" do
    get :index, {}, session_for(:inactive)
    assert_response :success
    assert_not_empty assigns(:required_user_agreements),
    "Inactive user did not have required_user_agreements"
    assert_template 'user_agreements/index',
    "Inactive user was not presented with a user agreement at the front page"
  end

  [[:active, true],
   [:project_viewer, false]].each do |which_user, should_show|
    test "create subproject button #{'not ' unless should_show} shown to #{which_user}" do
      readonly_project_uuid = api_fixture('groups')['aproject']['uuid']
      get :show, {
        id: readonly_project_uuid
      }, session_for(which_user)
      buttons = css_select('[data-method=post]').select do |el|
        el.attributes['href'].match /project.*owner_uuid.*#{readonly_project_uuid}/
      end
      if should_show
        assert_not_empty(buttons, "did not offer to create a subproject")
      else
        assert_empty(buttons.collect(&:to_s),
                     "offered to create a subproject in a non-writable project")
      end
    end
  end
end
