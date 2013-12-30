class UserAgreementsController < ApplicationController
  def model_class
    Collection
  end

  def sign
    params[:checked].each do |checked|
      if r = checked.match(/^([0-9a-f]+)/)
        UserAgreement.sign uuid: r[1]
      end
    end
    current_user.activate
    redirect_to(params[:return_to] || :back)
  end
end
