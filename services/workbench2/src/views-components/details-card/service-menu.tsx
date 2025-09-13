// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { Dispatch } from 'redux';
import { Button, Menu, MenuItem, Tooltip } from '@mui/material';
import { ExpandIcon } from 'components/icon/icon';
import { PublishedPort } from 'models/container';
import { showErrorSnackbar } from 'store/snackbar/snackbar-actions';

const mapDispatchToProps = (dispatch: Dispatch) => ({
    showErrorSnackbar: (message: string) => dispatch<any>(showErrorSnackbar(message)),
});

type ServiceMenuProps = {
    services: PublishedPort[];
    buttonClass?: string;
    showErrorSnackbar: (message: string) => void;
};

export const ServiceMenu = connect(
    null,
    mapDispatchToProps
)(({ services, buttonClass, showErrorSnackbar }: ServiceMenuProps) => {
    const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null);
    const open = Boolean(anchorEl);
    const handleOpen = (event: React.MouseEvent<HTMLButtonElement>) => {
        setAnchorEl(event.currentTarget);
    };
    const handleClose = () => {
        setAnchorEl(null);
    };
    const handleClick = (service: PublishedPort) => () => {
        handleClose();
        if (service.access === 'public') {
            window.open(service.initial_url, "_blank", "noopener");
        } else if (service.access === 'private') {
            // TODO Open initial_url with token
        } else {
            showErrorSnackbar("Published port access value not valid");
        }
    };

    if (services.length) {
        if (services.length === 1) {
            const service = services[0];

            return (
                <Tooltip arrow disableInteractive title={`Connect to ${service.label || "service"}`}>
                    <Button
                        className={buttonClass}
                        variant="contained"
                        size="small"
                        color="primary"
                        id="service-button"
                        onClick={handleClick(service)}
                    >
                        <span>Connect to {service.label || "service"}</span>
                    </Button>
                </Tooltip>
            );
        } else if (services.length > 1) {
            return <>
                <Button
                    className={buttonClass}
                    variant="contained"
                    size="small"
                    color="primary"
                    id="service-button"
                    aria-controls={open ? 'basic-menu' : undefined}
                    aria-haspopup="true"
                    aria-expanded={open ? 'true' : undefined}
                    onClick={handleOpen}
                    endIcon={<ExpandIcon />}
                >
                    <span>Connect to service</span>
                </Button>
                <Menu
                    id="basic-menu"
                    anchorEl={anchorEl}
                    open={open}
                    onClose={handleClose}
                    MenuListProps={{
                        'aria-labelledby': 'service-button',
                    }}
                >
                    {services.map((service: PublishedPort) => (
                        <MenuItem onClick={handleClick(service)}>
                            <span>{service.label}</span>
                        </MenuItem>
                    ))}
                </Menu>
            </>;
        }
    }

    // Return empty fragment when no services
    return <></>;
});
