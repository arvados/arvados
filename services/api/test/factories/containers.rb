# Read about factories at https://github.com/thoughtbot/factory_girl

FactoryGirl.define do
  factory :container do
    uuid "MyString"
    owner_uuid "MyString"
    created_at "MyString"
    modified_at "MyString"
    modified_by_client_uuid "MyString"
    modified_by_user_uuid "MyString"
    state "MyString"
    started_at "2015-12-02 10:14:26"
    finished_at "2015-12-02 10:14:26"
    log "MyString"
    environment "MyText"
    cwd "MyString"
    command "MyString"
    output_path "MyString"
    mounts "MyString"
    runtime_constraints "MyString"
    output "MyString"
    container_image "MyString"
    progress 1.5
    priority ""
  end
end
