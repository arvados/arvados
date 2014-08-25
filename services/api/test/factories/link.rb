FactoryGirl.define do
  factory :link do
    factory :permission_link do
      link_class 'permission'
    end
  end
end
