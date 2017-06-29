// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

(function() {
    var ok = false;
    try {
        if (window.Blob &&
            window.File &&
            window.FileReader &&
            window.localStorage &&
            window.WebSocket) {
            ok = true;
        }
    } catch(err) {}
    if (!ok) {
        document.getElementById('browser-unsupported').className='';
    }
})();
