// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Popover as MaterialPopover } from '@mui/material';

import { PopoverOrigin } from '@mui/material/Popover';
import IconButton, { IconButtonProps } from '@mui/material/IconButton';

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
                    data-cy="popover"
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
    <IconButton {...props} size="large">
        <i className="fas" />
    </IconButton>
);
