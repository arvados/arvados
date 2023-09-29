// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import Menu from '@material-ui/core/Menu';
import IconButton from '@material-ui/core/IconButton';
import { PopoverOrigin } from '@material-ui/core/Popover';
import { Tooltip } from '@material-ui/core';

interface DropdownMenuProps {
    id: string;
    icon: React.ReactElement<any>;
    title: string;
}

interface DropdownMenuState {
    anchorEl: any;
}

export class DropdownMenu extends React.Component<DropdownMenuProps, DropdownMenuState> {
    state = {
        anchorEl: undefined
    };

    transformOrigin: PopoverOrigin = {
        vertical: -50,
        horizontal: 0
    };

    render() {
        const { icon, id, children, title } = this.props;
        const { anchorEl } = this.state;
        return (
            <div>
                <Tooltip title={title}>
                    <IconButton
                        aria-owns={anchorEl ? id : undefined}
                        aria-haspopup="true"
                        color="inherit"
                        onClick={this.handleOpen}>
                        {icon}
                    </IconButton>
                </Tooltip>
                <Menu
                    id={id}
                    anchorEl={anchorEl}
                    open={Boolean(anchorEl)}
                    onClose={this.handleClose}
                    onClick={this.handleClose}
                    transformOrigin={this.transformOrigin}>
                    {children}
                </Menu>
            </div>
        );
    }

    handleClose = () => {
        this.setState({ anchorEl: undefined });
    }

    handleOpen = (event: React.MouseEvent<HTMLButtonElement>) => {
        this.setState({ anchorEl: event.currentTarget });
    }
}
