class UserAgreementsController < ApplicationController
  skip_before_filter :check_user_agreements
  skip_before_filter :find_object_by_uuid
  skip_before_filter :check_user_profile

  def index
    if unsigned_user_agreements.empty?
      redirect_to(params[:return_to] || :back)
    end
  end

  def model_class
    Collection
  end

  def sign
    params[:checked].each do |checked|
      if (r = CollectionsHelper.match_uuid_with_optional_filepath(checked))
        UserAgreement.sign uuid: r[1]
      end
    end
    current_user.activate
    redirect_to(params[:return_to] || :back)
  end
end
