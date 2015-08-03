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
      assert page.has_text?('Please log in'), 'Not found text - Please log in'
    elsif user['is_active']
      if profile_config && !has_profile
        assert page.has_text?('Save profile'), 'No text - Save profile'
        add_profile user
      else
        assert page.has_text?('Active pipelines'), 'Not found text - Active pipelines'
        assert page.has_no_text?('Save profile'), 'Found text - Save profile'
      end
    elsif invited
      assert page.has_text?('Please check the box below to indicate that you have read and accepted the user agreement'),
        'Not found text - Please check the box below . . .'
      assert page.has_no_text?('Save profile'), 'Found text - Save profile'
    else
      assert page.has_text?('Your account is inactive'), 'Not found text - Your account is inactive'
      assert page.has_no_text?('Save profile'), 'Found text - Save profile'
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
        assert(page.has_link?("notifications-menu"), 'no user menu')
        page.find("#notifications-menu").click
        within('.dropdown-menu') do
          if user['is_active']
            assert page.has_no_link?('Not active'), 'Found link - Not active'
            assert page.has_no_link?('Sign agreements'), 'Found link - Sign agreements'

            assert page.has_link?('Virtual machines'), 'No link - Virtual machines'
            assert page.has_link?('Repositories'), 'No link - Repositories'
            assert page.has_link?('Current token'), 'No link - Current token'
            assert page.has_link?('SSH keys'), 'No link - SSH Keys'

            if profile_config
              assert page.has_link?('Manage profile'), 'No link - Manage profile'
            else
              assert page.has_no_link?('Manage profile'), 'Found link - Manage profile'
            end
          end
          assert page.has_link?('Log out'), 'No link - Log out'
        end
      end
    end
  end

  # Check manage profile page and add missing profile to the user
  def add_profile user
    assert page.has_no_text?('My projects'), 'Found text - My projects'
    assert page.has_no_text?('Projects shared with me'), 'Found text - Projects shared with me'

    assert page.has_text?('Profile'), 'No text - Profile'
    assert page.has_text?('First Name'), 'No text - First Name'
    assert page.has_text?('Last Name'), 'No text - Last Name'
    assert page.has_text?('Identity URL'), 'No text - Identity URL'
    assert page.has_text?('E-mail'), 'No text - E-mail'
    assert page.has_text?(user['email']), 'No text - user email'

    # Using the default profile which has message and one required field

    # Save profile without filling in the required field. Expect to be back in this profile page again
    click_button "Save profile"
    assert page.has_text?('Profile'), 'No text - Profile'
    assert page.has_text?('First Name'), 'No text - First Name'
    assert page.has_text?('Last Name'), 'No text - Last Name'
    assert page.has_text?('Save profile'), 'No text - Save profile'

    # This time fill in required field and then save. Expect to go to requested page after that.
    profile_message = Rails.configuration.user_profile_form_message
    required_field_title = ''
    required_field_key = ''
    profile_config = Rails.configuration.user_profile_form_fields
    profile_config.andand.each do |entry|
      if entry['required']
        required_field_key = entry['key']
        required_field_title = entry['form_field_title']
      end
    end

    assert page.has_text? profile_message.gsub(/<.*?>/,'')
    assert page.has_text?(required_field_title), 'No text - configured required field title'

    page.find_field('user[prefs][:profile][:'+required_field_key+']').set 'value to fill required field'

    click_button "Save profile"
    # profile saved and in profile page now with success
    assert page.has_text?('Thank you for filling in your profile'), 'No text - Thank you for filling'
    if user['prefs']['getting_started_shown']
      click_link 'Back to work!'
    else
      click_link 'Get started'
    end

    # profile saved and in home page now
    assert page.has_text?('Active pipelines'), 'No text - Active pipelines'
  end

  [
    [nil, nil, false, false],
    ['inactive', api_fixture('users')['inactive'], true, false],
    ['inactive_uninvited', api_fixture('users')['inactive_uninvited'], false, false],
    ['active', api_fixture('users')['active'], true, true],
    ['admin', api_fixture('users')['admin'], true, true],
    ['active_no_prefs', api_fixture('users')['active_no_prefs'], true, false],
    ['active_no_prefs_profile_no_getting_started_shown',
      api_fixture('users')['active_no_prefs_profile_no_getting_started_shown'], true, false],
    ['active_no_prefs_profile_with_getting_started_shown',
      api_fixture('users')['active_no_prefs_profile_with_getting_started_shown'], true, false],
  ].each do |token, user, invited, has_profile|

    test "visit home page when profile is configured for user #{token}" do
      # Our test config enabled profile by default. So, no need to update config
      if !token
        visit ('/')
      else
        visit page_with_token(token)
      end

      verify_homepage_with_profile user, invited, has_profile
    end

    test "visit home page when profile not configured for user #{token}" do
      Rails.configuration.user_profile_form_fields = false

      if !token
        visit ('/')
      else
        visit page_with_token(token)
      end

      verify_homepage_with_profile user, invited, has_profile
    end

  end

end
