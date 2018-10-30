// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';

export interface AdvancedViewSwitchInjectedProps {
    toggleAdvancedView: () => void;
    advancedViewOpen: boolean;
}

export const connectAdvancedViewSwitch = (Component: React.ComponentType<AdvancedViewSwitchInjectedProps>) =>
    class extends React.Component<{}, { advancedViewOpen: boolean }> {

        state = { advancedViewOpen: false };

        toggleAdvancedView = () => {
            this.setState(({ advancedViewOpen }) => ({ advancedViewOpen: !advancedViewOpen }));
        }

        render() {
            return <Component {...this.state} {...this} />;
        }
    };
    