// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import classNames from 'classnames';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Card, CardHeader, Typography, Grid, Button, Menu, MenuItem } from '@mui/material';
import { StartIcon, StopIcon, ExpandIcon } from 'components/icon/icon';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { connect } from 'react-redux';
import { MultiselectToolbar } from 'components/multiselect-toolbar/MultiselectToolbar';
import { RootState } from 'store/store';
import { Dispatch } from 'redux';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';
import { setSelectedResourceUuid } from 'store/selected-resource/selected-resource-actions';
import { deselectAllOthers } from 'store/multiselect/multiselect-actions';
import { isProcessCancelable, isProcessRunnable, isProcessResumable, isProcessRunning } from 'store/processes/process';
import { ProcessStatus } from 'views-components/data-explorer/renderers';
import { cancelRunningWorkflow, resumeOnHoldWorkflow, startWorkflow } from 'store/processes/processes-actions';
import { Process } from 'store/processes/process';
import { getProcess } from 'store/processes/process';
import { PublishedPort } from 'models/container';

type CssRules = 'root' | 'cardHeaderContainer' | 'cardHeader' | 'nameContainer' | 'buttonContainer' | 'runStatusContainer' | 'runStatusContainerWithServiceButton' | 'actionButton' | 'runButton' | 'cancelButton' | 'serviceButton' | 'toolbarStyles';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        marginBottom: '1rem',
        flex: '0 0 auto',
        padding: 0,
        minHeight: '3rem',
    },
    nameContainer: {
        display: 'flex',
        alignItems: 'center',
        minHeight: '2.7rem',
        gap: '2rem',
    },
    cardHeaderContainer: {
        width: '100%',
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'flex-start',
        justifyContent: 'space-between',
    },
    cardHeader: {
        minWidth: '30rem',
        padding: '0.2rem 0.4rem 0.2rem 1rem',
        '& > div': {
            overflow: "hidden",
        },
    },
    buttonContainer: {
        overflow: 'hidden',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'flex-start',
        rowGap: '5px',
        flexWrap: 'wrap',
        flexGrow: 0,
        flexBasis: '200px',
        minWidth: '200px',
    },
    runStatusContainer: {
        width: '100%',
        display: 'flex',
        columnGap: '5px',

    },
    // Only active when service button is shown
    runStatusContainerWithServiceButton: {
        '& > *': {
            // Allow run/cancel status to share space
            flexGrow: 1,
            flexShrink: 1,
        },
    },
    actionButton: {
        padding: "0px 5px 0 0",
        fontSize: '0.78rem',
        // Set icon size for all buttons
        '& svg': {
            fontSize: '22px',
        },
        whiteSpace: 'nowrap',
    },
    runButton: {
        flexShrink: 0,
    },
    cancelButton: {
        flexShrink: 0,
        color: theme.palette.common.white,
        backgroundColor: theme.customs.colors.red900,
        '&:hover': {
            backgroundColor: theme.customs.colors.red900,
        },
    },
    serviceButton: {
        width: '100%',
        // Add padding to account for no icon
        paddingLeft: '5px',
        justifyContent: 'center',
        '& span': {
            // Ellipse button text
            overflow: 'hidden',
            textOverflow: 'ellipsis',
        },
    },
    toolbarStyles: {
        paddingTop: '4px',
    },
});

const mapStateToProps = ({ auth, selectedResource, resources, properties }: RootState) => {
    const currentResource = getProcess(properties.currentRouteUuid)(resources);
    const isSelected = selectedResource.selectedResourceUuid === properties.currentRouteUuid;

    return {
        isAdmin: auth.user?.isAdmin,
        currentResource,
        isSelected,
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    handleCardClick: (uuid: string) => {
        dispatch<any>(loadDetailsPanel(uuid));
        dispatch<any>(setSelectedResourceUuid(uuid));
        dispatch<any>(deselectAllOthers(uuid));
    },
    cancelProcess: (uuid: string) => dispatch<any>(cancelRunningWorkflow(uuid)),
    startProcess: (uuid: string) => dispatch<any>(startWorkflow(uuid)),
    resumeOnHoldWorkflow: (uuid: string) => dispatch<any>(resumeOnHoldWorkflow(uuid)),
});

type ProcessCardProps = WithStyles<CssRules> & {
    currentResource: Process;
    isSelected: boolean;
    handleCardClick: (resource: any) => void;
    cancelProcess: (uuid: string) => void;
    startProcess: (uuid: string) => void;
    resumeOnHoldWorkflow: (uuid: string) => void;
};

export const ProcessCard = connect(
    mapStateToProps,
    mapDispatchToProps
)(
    withStyles(styles)((props: ProcessCardProps) => {
        const { classes, currentResource, handleCardClick, isSelected , cancelProcess, startProcess, resumeOnHoldWorkflow } = props;
        const { name, uuid } = currentResource.containerRequest;
        let publishedPorts: PublishedPort[] = [];

        if (currentResource.container && currentResource.container.publishedPorts) {
            const ports = currentResource.container.publishedPorts;
            publishedPorts = Object.keys(ports).map((port: string) => (ports[port]));
        }

        const showServiceMenu: boolean = isProcessRunning(currentResource) && !!publishedPorts.length;

        let runAction;
        if (isProcessRunnable(currentResource)) {
            runAction = startProcess;
        } else if (isProcessResumable(currentResource)) {
            runAction = resumeOnHoldWorkflow;
        }

        return (
            <Card
                className={classes.root}
                onClick={() => handleCardClick(uuid)}
                data-cy='process-details-card'
            >
                <Grid
                    container
                    wrap='nowrap'
                    className={classes.cardHeaderContainer}
                >
                    <CardHeader
                        className={classes.cardHeader}
                        title={
                            <section className={classes.nameContainer}>
                                <Typography
                                    variant='h6'
                                >
                                    {name}
                                </Typography>
                                <section className={classes.buttonContainer}>
                                    {showServiceMenu && <ServiceMenu buttonClass={classNames(classes.actionButton, classes.serviceButton)} services={publishedPorts} />}
                                    <div className={classNames(classes.runStatusContainer, showServiceMenu ? classes.runStatusContainerWithServiceButton : undefined)}>
                                        {runAction !== undefined &&
                                            <Button
                                                data-cy="process-run-button"
                                                variant="contained"
                                                size="small"
                                                color="primary"
                                                className={classNames(classes.actionButton, classes.runButton)}
                                                onClick={() => runAction && runAction(currentResource.containerRequest.uuid)}>
                                                <StartIcon />
                                                Run
                                            </Button>}
                                        {isProcessCancelable(currentResource) &&
                                            <Button
                                                data-cy="process-cancel-button"
                                                variant="contained"
                                                size="small"
                                                color="primary"
                                                className={classNames(classes.actionButton, classes.cancelButton)}
                                                onClick={() => cancelProcess(currentResource.containerRequest.uuid)}>
                                                <StopIcon />
                                                Cancel
                                            </Button>}
                                        <ProcessStatus uuid={currentResource.containerRequest.uuid} />
                                    </div>
                                </section>
                            </section>
                        }
                    />
                    {isSelected && <MultiselectToolbar injectedStyles={classes.toolbarStyles} />}
                </Grid>
            </Card>
        );
    })
);

type ServiceMenuProps = {
    services: PublishedPort[];
    buttonClass?: string;
};

const ServiceMenu = ({ services, buttonClass }: ServiceMenuProps) => {
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
        window.open(service.initial_url, "_blank", "noopener");
    };

    if (services.length) {
        if (services.length === 1) {
            const service = services[0];

            return (
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
  }
