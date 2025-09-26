// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { Dispatch } from 'redux';
import { Button, Menu, MenuItem, Tooltip } from '@mui/material';
import { ExpandIcon } from 'components/icon/icon';
import { PublishedPort } from 'models/container';
import { showErrorSnackbar } from 'store/snackbar/snackbar-actions';
import { injectTokenParam } from 'common/url';

const mapStateToProps = ({ auth }: RootState) => ({
    userToken: auth.apiToken,
});

const mapDispatchToProps = (dispatch: Dispatch) => ({
    showErrorSnackbar: (message: string) => dispatch<any>(showErrorSnackbar(message)),
});

type ServiceMenuProps = {
    services: PublishedPort[];
    buttonClass?: string;
    userToken: string | undefined;
    showErrorSnackbar: (message: string) => void;
};

export const ServiceMenu = connect(
    mapStateToProps,
    mapDispatchToProps
)(({ services, buttonClass, userToken, showErrorSnackbar }: ServiceMenuProps) => {
    const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null);
    const open = Boolean(anchorEl);
    const handleOpen = (event: React.MouseEvent<HTMLButtonElement>) => {
        setAnchorEl(event.currentTarget);
    };
    const handleClose = () => {
        setAnchorEl(null);
    };
    const handleClick = (service: PublishedPort) => async () => {
        handleClose();

        if (!service.initial_url) {
            showErrorSnackbar("Service URL not set");
            return;
        }

        if (service.access === 'public') {
            // Open public links as-is
            window.open(service.initial_url, "_blank", "noopener");
        } else if (service.access === 'private') {
            // Open private links with user token
            if (userToken) {
                try {
                    const url = await injectTokenParam(service.initial_url, userToken);
                    window.open(url, "_blank", "noopener");
                } catch(e) {
                    showErrorSnackbar("Failed to open service: " + e.message);
                }
            } else {
                showErrorSnackbar("User token not available");
            }
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
                    aria-controls={open ? 'service-menu' : undefined}
                    aria-haspopup="true"
                    aria-expanded={open ? 'true' : undefined}
                    onClick={handleOpen}
                    endIcon={<ExpandIcon />}
                >
                    <span>Connect to service</span>
                </Button>
                <Menu
                    id="service-menu"
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
