// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";

export const useAsyncInterval = function (callback, delay) {
    const ref = React.useRef<{cb: () => Promise<any>, active: boolean}>({
        cb: async () => {},
        active: false}
    );

    // Remember the latest callback.
    React.useEffect(() => {
        ref.current.cb = callback;
    }, [callback]);
    // Set up the interval.
    React.useEffect(() => {
        function tick() {
            if (ref.current.active) {
                // Wrap execution chain with promise so that execution errors or
                //   non-async callbacks still fall through to .finally, avoids breaking polling
                new Promise((resolve) => {
                    return resolve(ref.current.cb());
                }).then(() => {
                    // Promise succeeded
                    // Possibly implement back-off reset
                }).catch(() => {
                    // Promise rejected
                    // Possibly implement back-off in the future
                }).finally(() => {
                    setTimeout(tick, delay);
                });
            }
        }
        if (delay !== null) {
            ref.current.active = true;
            setTimeout(tick, 0); // want the first callback to happen immediately.
        }
        // Suppress warning about cleanup function - can be ignored when variables are unrelated to dom elements
        //   https://github.com/facebook/react/issues/15841#issuecomment-500133759
        // eslint-disable-next-line
        return () => {ref.current.active = false;};
    }, [delay]);
};
