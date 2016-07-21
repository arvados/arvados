require 'integration_helper'

class UserProfileTest < ActionDispatch::IntegrationTest
  setup do
    need_javascript
    @user_profile_form_fields = Rails.configuration.user_profile_form_fields
  end

  teardown do
    Rails.configuration.user_profile_form_fields = @user_profile_form_fields
  end

  def verify_homepage_with_profile user, invited, has_profile
    profile_config = Rails.configuration.user_profile_form_fields

    if !user
      assert_text('Please log in')
    elsif user['is_active']
      if profile_config && !has_profile
        assert_text('Save profile')
        add_profile user
      else
        assert_text('Recent pipelines and processes')
        assert_no_text('Save profile')
      end
    elsif invited
      assert_text('Please check the box below to indicate that you have read and accepted the user agreement')
      assert_no_text('Save profile')
    else
      assert_text('Your account is inactive')
      assert_no_text('Save profile')
    end

    # If the user has not already seen getting_started modal, it will be shown on first visit.
    if user and user['is_active'] and !user['prefs']['getting_started_shown']
      within '.modal-content' do
        assert_text 'Getting Started'
        assert_selector 'button', text: 'Next'
        assert_selector 'button', text: 'Prev'
        first('button', text: 'x').click
      end
    end

    within('.navbar-fixed-top') do
      if !user
        assert page.has_link?('Log in'), 'Not found link - Log in'
      else
        # my account menu
        assert_selector("#notifications-menu")
        page.find("#notifications-menu").click
        within('.dropdown-menu') do
          if user['is_active']
            assert_no_selector('a', text: 'Not active')
            assert_no_selector('a', text: 'Sign agreements')

            assert_selector('a', text: 'Virtual machines')
            assert_selector('a', text: 'Repositories')
            assert_selector('a', text: 'Current token')
            assert_selector('a', text: 'SSH keys')

            if profile_config
              assert_selector('a', text: 'Manage profile')
            else
              assert_no_selector('a', text: 'Manage profile')
            end
          end
          assert_selector('a', text: 'Log out')
        end
      end
    end
  end

  # Check manage profile page and add missing profile to the user
  def add_profile user
    assert_no_text('My projects')
    assert_no_text('Projects shared with me')

    assert_text('Profile')
    assert_text('First Name')
    assert_text('Last Name')
    assert_text('Identity URL')
    assert_text('E-mail')
    assert_text(user['email'])

    # Using the default profile which has message and one required field

    # Save profile without filling in the required field. Expect to be back in this profile page again
    click_button "Save profile"
    assert_text('Profile')
    assert_text('First Name')
    assert_text('Last Name')
    assert_text('Save profile')

    # This time fill in required field and then save. Expect to go to requested page after that.
    profile_message = Rails.configuration.user_profile_form_message
    required_field_title = ''
    required_field_key = ''
    profile_config = Rails.configuration.user_profile_form_fields
    profile_config.each do |entry|
      if entry['required']
        required_field_key = entry['key']
        required_field_title = entry['form_field_title']
        break
      end
    end

    assert page.has_text? profile_message.gsub(/<.*?>/,'')
    assert_text(required_field_title)

    page.find_field('user[prefs][profile]['+required_field_key+']').set 'value to fill required field'

    click_button "Save profile"
    # profile saved and in profile page now with success
    assert_text('Thank you for filling in your profile')
    assert_selector('input' +
                    '[name="user[prefs][profile]['+required_field_key+']"]' +
                    '[value="value to fill required field"]')
    if user['prefs']['getting_started_shown']
      click_link 'Back to work!'
    else
      click_link 'Get started'
    end

    # profile saved and in home page now
    assert_text('Recent pipelines and processes')
  end

  [
    [nil, false, false],
    ['inactive', true, false],
    ['inactive_uninvited', false, false],
    ['active', true, true],
    ['admin', true, true],
    ['active_no_prefs', true, false],
    ['active_no_prefs_profile_no_getting_started_shown', true, false],
    ['active_no_prefs_profile_with_getting_started_shown', true, false],
  ].each do |token, invited, has_profile|
    [true, false].each do |profile_required|
      test "visit #{token} home page when profile is #{'not ' if !profile_required}configured" do
        if !profile_required
          Rails.configuration.user_profile_form_fields = false
        else
          # Our test config enabled profile by default. So, no need to update config
        end
        Rails.configuration.enable_getting_started_popup = true

        if !token
          visit ('/')
        else
          visit page_with_token(token)
        end

        user = token && api_fixture('users')[token]
        verify_homepage_with_profile user, invited, has_profile
      end
    end
  end
end
