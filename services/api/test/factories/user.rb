include CurrentApiClient

FactoryGirl.define do
  factory :user do
    before :create do
      Thread.current[:user_was] = Thread.current[:user]
      Thread.current[:user] = system_user
    end
    after :create do
      Thread.current[:user] = Thread.current[:user_was]
    end
    first_name "Factory"
    last_name "Factory"
    identity_url do
      "https://example.com/#{rand(2**24).to_s(36)}"
    end
    factory :active_user do
      is_active true
      after :create do |user|
        act_as_system_user do
          Link.create!(tail_uuid: user.uuid,
                       head_uuid: Group.where('uuid ~ ?', '-f+$').first.uuid,
                       link_class: 'permission',
                       name: 'can_read')
        end
      end
    end
  end
end
