class UserAgreementsController < ApplicationController
  skip_before_filter :check_user_agreements
  skip_before_filter :find_object_by_uuid
  skip_before_filter :check_user_profile

  def model_class
    Collection
  end

  def sign
    params[:checked].each do |checked|
      if r = checked.match(/^([0-9a-f]+[^\/]*)/)
        UserAgreement.sign uuid: r[1]
      end
    end
    current_user.activate
    redirect_to(params[:return_to] || :back)
  end
end
