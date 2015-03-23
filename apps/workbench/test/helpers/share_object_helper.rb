module ShareObjectHelper
  def show_object_using(auth_key, type, key, expect)
    obj_uuid = api_fixture(type)[key]['uuid']
    visit(page_with_token(auth_key, "/#{type}/#{obj_uuid}"))
    assert(page.has_text?(expect), "expected string not found: #{expect}")
  end

  def share_rows
    find('#object_sharing').all('tr')
  end

  def add_share_and_check(share_type, name, obj=nil)
    assert(page.has_no_text?(name), "project is already shared with #{name}")
    start_share_count = share_rows.size
    click_on("Share with #{share_type}")
    within(".modal-container") do
      # Order is important here: we should find something that appears in the
      # modal before we make any assertions about what's not in the modal.
      # Otherwise, the not-included assertions might falsely pass because
      # the modal hasn't loaded yet.
      find(".selectable", text: name).click
      assert(has_no_selector?(".modal-dialog-preview-pane"),
             "preview pane available in sharing dialog")
      if share_type == 'users' and obj and obj['email']
        assert(page.has_text?(obj['email']), "Did not find user's email")
      end
      assert_raises(Capybara::ElementNotFound,
                    "Projects pulldown available from sharing dialog") do
        click_on "All projects"
      end
      click_on "Add"
    end
    using_wait_time(Capybara.default_wait_time * 3) do
      assert(page.has_link?(name),
             "new share was not added to sharing table")
      assert_equal(start_share_count + 1, share_rows.size,
                   "new share did not add row to sharing table")
    end
  end

  def modify_share_and_check(name)
    start_rows = share_rows
    link_row = start_rows.select { |row| row.has_text?(name) }
    assert_equal(1, link_row.size, "row with new permission not found")
    within(link_row.first) do
      click_on("Read")
      select("Write", from: "share_change_level")
      click_on("editable-submit")
      assert(has_link?("Write"),
             "failed to change access level on new share")
      click_on "Revoke"
      if Capybara.current_driver == :selenium
        page.driver.browser.switch_to.alert.accept
      else
        # poltergeist returns true for confirm(), so we don't need to accept.
      end
    end
    # Ensure revoked permission disappears from page.
    using_wait_time(Capybara.default_wait_time * 3) do
      assert_no_text name
      assert_equal(start_rows.size - 1, share_rows.size,
                   "revoking share did not remove row from sharing table")
    end
  end

  def user_can_manage(user_sym, fixture)
    get(:show, {id: fixture["uuid"]}, session_for(user_sym))
    is_manager = assigns(:user_is_manager)
    assert_not_nil(is_manager, "user_is_manager flag not set")
    if not is_manager
      assert_empty(assigns(:share_links),
                   "non-manager has share links set")
    end
    is_manager
  end

end
