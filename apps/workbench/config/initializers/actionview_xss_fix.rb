# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# This is related to:
# * https://github.com/advisories/GHSA-65cv-r6x7-79hv
# * https://nvd.nist.gov/vuln/detail/CVE-2020-5267
#
# Until we upgrade to rails 5.2, this monkeypatch should be enough
ActionView::Helpers::JavaScriptHelper::JS_ESCAPE_MAP.merge!(
  {
    "`" => "\\`",
    "$" => "\\$"
  }
)

module ActionView::Helpers::JavaScriptHelper
  alias :old_ej :escape_javascript
  alias :old_j :j

  def escape_javascript(javascript)
    javascript = javascript.to_s
    if javascript.empty?
      result = ""
    else
      result = javascript.gsub(/(\\|<\/|\r\n|\342\200\250|\342\200\251|[\n\r"']|[`]|[$])/u, JS_ESCAPE_MAP)
    end
    javascript.html_safe? ? result.html_safe : result
  end

  alias :j :escape_javascript
end