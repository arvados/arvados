# Read about factories at https://github.com/thoughtbot/factory_girl

FactoryGirl.define do
  factory :container_request do
    uuid "MyString"
    owner_uuid "MyString"
    created_at "2015-12-02 10:17:04"
    modified_at "2015-12-02 10:17:04"
    modified_by_client_uuid "MyString"
    modified_by_user_uuid "MyString"
    name "MyString"
    description "MyText"
    properties "MyString"
    state "MyString"
    requesting_container_uuid "MyString"
    container_uuid "MyString"
    container_count_max ""
    mounts "MyString"
    runtime_constraints "MyString"
    container_image "MyString"
    environment "MyString"
    cwd "MyString"
    command "MyString"
    output_path "MyString"
    priority ""
    expires_at "2015-12-02 10:17:04"
    filters "MyString"
  end
end
