FactoryGirl.define do
  factory :api_client do
    is_trusted false
    to_create do |instance|
      act_as_system_user do
        instance.save!
      end
    end
  end
end
