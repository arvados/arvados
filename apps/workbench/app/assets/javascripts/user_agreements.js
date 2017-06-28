// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

function enable_okbutton() {
    var $div = $('#open_user_agreement');
    var allchecked = $('input[name="checked[]"]', $div).not(':checked').length == 0;
    $('input[type=submit]', $div).prop('disabled', !allchecked);
}
$(document).on('click keyup input', '#open_user_agreement input', enable_okbutton);
$(document).on('ready ajax:complete', enable_okbutton);
