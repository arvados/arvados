// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

var react = require("react");

export const useAsyncInterval = function (callback, delay) {
    const savedCallback = react.useRef();
    const active = react.useRef(false);

    // Remember the latest callback.
    react.useEffect(() => {
        savedCallback.current = callback;
    }, [callback]);
    // Set up the interval.
    react.useEffect(() => {
        // useEffect doesn't like async callbacks (https://github.com/facebook/react/issues/14326) so create nested async callback
        (async () => {
            // Make tick() async
            async function tick() {
                if (active.current) {
                    // If savedCallback is not set yet, no-op until it is
                    savedCallback.current && await savedCallback.current();
                    setTimeout(tick, delay);
                }
            }
            if (delay !== null) {
                active.current = true;
                setTimeout(tick, delay);
            }
        })(); // Call nested async function
        // We return the teardown function here since we can't from inside the nested async callback
        return () => {active.current = false;};
    }, [delay]);
};
