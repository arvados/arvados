class UserAgreementsController < ApplicationController
  skip_before_filter :check_user_agreements
  skip_before_filter :find_object_by_uuid

  def index
    @required_user_agreements = []
    @signed_user_agreements = []
    signed_ua_uuids = UserAgreement.signatures.map &:head_uuid
    UserAgreement.all.each do |ua|
      ua_collection = Collection.find(ua.uuid)
      if signed_ua_uuids.index(ua.uuid)
        @signed_user_agreements << ua_collection
      else
        @required_user_agreements << ua_collection
      end
    end

    super
  end

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
