class ApiClientAuthorization < ActiveRecord::Base
  belongs_to :api_client
  belongs_to :user
  after_initialize :assign_random_api_token

  def assign_random_api_token
    self.api_token ||= rand(2**256).to_s(36)
  end
end
