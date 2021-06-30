// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Popover as MaterialPopover } from '@material-ui/core';

import { PopoverOrigin } from '@material-ui/core/Popover';
import IconButton, { IconButtonProps } from '@material-ui/core/IconButton';

export interface PopoverProps {
    triggerComponent?: React.ComponentType<{ onClick: (event: React.MouseEvent<any>) => void }>;
    closeOnContentClick?: boolean;
}

export class Popover extends React.Component<PopoverProps> {
    state = {
        anchorEl: undefined
    };

    transformOrigin: PopoverOrigin = {
        vertical: "top",
        horizontal: "right",
    };

    render() {
        const Trigger = this.props.triggerComponent || DefaultTrigger;
        return (
            <>
                <Trigger onClick={this.handleTriggerClick} />
                <MaterialPopover
                    anchorEl={this.state.anchorEl}
                    open={Boolean(this.state.anchorEl)}
                    onClose={this.handleClose}
                    onClick={this.handleSelfClick}
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

    handleTriggerClick = (event: React.MouseEvent<any>) => {
        this.setState({ anchorEl: event.currentTarget });
    }

    handleSelfClick = () => {
        if (this.props.closeOnContentClick) {
            this.handleClose();
        }
    }
}

export const DefaultTrigger: React.SFC<IconButtonProps> = (props) => (
    <IconButton {...props}>
        <i className="fas" />
    </IconButton>
);
