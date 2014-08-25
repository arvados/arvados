FactoryGirl.define do
  factory :api_client_authorization do
    api_client
    scopes ['all']

    trait :trusted do
      association :api_client, factory: :api_client, is_trusted: true
    end
    factory :token do
      # Just provides shorthand for "create :api_client_authorization"
    end

    to_create do |instance|
      act_as_user instance.user do
        instance.save!
      end
    end
  end
end
