require 'test_helper'

class Arvados::V1::LinksControllerTest < ActionController::TestCase

  test "no symbol keys in serialized hash" do
    link = {
      properties: {username: 'testusername'},
      link_class: 'test',
      name: 'encoding',
      tail_kind: 'arvados#user',
      tail_uuid: users(:admin).uuid,
      head_kind: 'arvados#virtualMachine',
      head_uuid: virtual_machines(:testvm).uuid
    }
    authorize_with :admin
    [link, link.to_json].each do |formatted_link|
      post :create, link: formatted_link
      assert_response :success
      assert_not_nil assigns(:object)
      assert_equal 'testusername', assigns(:object).properties['username']
      assert_equal false, assigns(:object).properties.has_key?(:username)
    end
  end

  %w(created_at updated_at modified_at).each do |attr|
    {nil: nil, bogus: 2.days.ago}.each do |bogustype, bogusvalue|
      test "cannot set #{bogustype} #{attr} in create" do
        authorize_with :active
        post :create, {
          link: {
            properties: {},
            link_class: 'test',
            name: 'test',
          }.merge(attr => bogusvalue)
        }
        assert_response :success
        resp = JSON.parse @response.body
        assert_in_delta Time.now, Time.parse(resp[attr]), 3.0
      end
      test "cannot set #{bogustype} #{attr} in update" do
        really_created_at = links(:test_timestamps).created_at
        authorize_with :active
        put :update, {
          id: links(:test_timestamps).uuid,
          link: {
            :properties => {test: 'test'},
            attr => bogusvalue
          }
        }
        assert_response :success
        resp = JSON.parse @response.body
        case attr
        when 'created_at'
          assert_in_delta really_created_at, Time.parse(resp[attr]), 0.001
        else
          assert_in_delta Time.now, Time.parse(resp[attr]), 3.0
        end
      end
    end
  end
end
