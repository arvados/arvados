# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module ArvadosApiClientHelper
  def arvados_api_client
    ArvadosApiClient.new_or_current
  end
end

# For the benefit of themes that still expect $arvados_api_client to work:
class ArvadosClientProxyHack
  def method_missing *args
    ArvadosApiClient.new_or_current.send *args
  end
end
$arvados_api_client = ArvadosClientProxyHack.new
