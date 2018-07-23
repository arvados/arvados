// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import Menu from '@material-ui/core/Menu';
import IconButton from '@material-ui/core/IconButton';
import { PopoverOrigin } from '@material-ui/core/Popover';

interface DropdownMenuProps {
    id: string;
    icon: React.ReactElement<any>;
}

interface DropdownMenuState {
    anchorEl: any;
}

export class DropdownMenu extends React.Component<DropdownMenuProps, DropdownMenuState> {
    state = {
        anchorEl: undefined
    };

    transformOrigin: PopoverOrigin = {
        vertical: "top",
        horizontal: "center"
    };

    render() {
        const { icon, id, children } = this.props;
        const { anchorEl } = this.state;
        return (
            <div>
                <IconButton
                    aria-owns={anchorEl ? id : undefined}
                    aria-haspopup="true"
                    color="inherit"
                    onClick={this.handleOpen}>
                    {icon}
                </IconButton>
                <Menu
                    id={id}
                    anchorEl={anchorEl}
                    open={Boolean(anchorEl)}
                    onClose={this.handleClose}
                    onClick={this.handleClose}
                    anchorOrigin={this.transformOrigin}
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
