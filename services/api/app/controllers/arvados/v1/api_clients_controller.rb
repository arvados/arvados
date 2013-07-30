class Arvados::V1::ApiClientsController < ApplicationController
  before_filter :admin_required
end
