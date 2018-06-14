// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Popover as MaterialPopover } from '@material-ui/core';

import { PopoverOrigin } from '@material-ui/core/Popover';

export interface PopoverProps {
    triggerComponent: React.ComponentType<{ onClick: (event: React.MouseEvent<any>) => void }>;
}


class Popover extends React.Component<PopoverProps> {

    state = {
        anchorEl: undefined
    };

    transformOrigin: PopoverOrigin = {
        vertical: "top",
        horizontal: "right",
    };

    render() {
        return (
            <>
                <this.props.triggerComponent onClick={this.handleClick} />
                <MaterialPopover
                    anchorEl={this.state.anchorEl}
                    open={Boolean(this.state.anchorEl)}
                    onClose={this.handleClose}
                    transformOrigin={this.transformOrigin}
                    anchorOrigin={this.transformOrigin}
                >
                    {this.props.children}
                </MaterialPopover>
            </>
        );
    }

    handleClose = () => {
        this.setState({ anchorEl: undefined });
    }

    handleClick = (event: React.MouseEvent<any>) => {
        this.setState({ anchorEl: event.currentTarget });
    }

}

export default Popover;
