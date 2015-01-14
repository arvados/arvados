require 'integration_helper'
require 'helpers/share_object_helper'

class RepositoriesTest < ActionDispatch::IntegrationTest
  include ShareObjectHelper

  setup do
    need_javascript
  end

  [
    'active', #owner
    'admin'
  ].each do |user|
    test "#{user} can manage sharing for another user" do
      add_user = api_fixture('users')['future_project_user']
      new_name = ["first_name", "last_name"].map { |k| add_user[k] }.join(" ")

      show_repository_using(user, 'foo')
      click_on "Sharing"
      add_share_and_check("users", new_name, add_user)
      modify_share_and_check(new_name)
    end
  end

  [
    'active', #owner
    'admin'
  ].each do |user|
    test "#{user} can manage sharing for another group" do
      new_name = api_fixture('groups')['future_project_viewing_group']['name']

      show_repository_using("active", 'foo')
      click_on "Sharing"
      add_share_and_check("groups", new_name)
      modify_share_and_check(new_name)
    end
  end

  test "spectator does not see repository sharing tab" do
    show_repository_using("spectator")
    assert(page.has_no_link?("Sharing"),
           "read-only repository user sees sharing tab")
  end
end
