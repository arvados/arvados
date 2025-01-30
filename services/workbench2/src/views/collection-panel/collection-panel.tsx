// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { IconButton, Grid, Tooltip, Typography, Card, CardHeader, CardContent } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { connect, DispatchProp } from "react-redux";
import { RouteComponentProps } from 'react-router';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { MoreVerticalIcon, CollectionIcon, ReadOnlyIcon, CollectionOldVersionIcon } from 'components/icon/icon';
import { DetailsAttribute } from 'components/details-attribute/details-attribute';
import { CollectionResource, getCollectionUrl } from 'models/collection';
import { CollectionPanelFiles } from 'views-components/collection-panel-files/collection-panel-files';
import { navigateToProcess } from 'store/collection-panel/collection-panel-action';
import { ResourcesState } from 'store/resources/resources';
import { openContextMenu, resourceUuidToContextMenuKind } from 'store/context-menu/context-menu-actions';
import { formatDate, formatFileSize } from "common/formatters";
import { openDetailsPanel } from 'store/details-panel/details-panel-action';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { getPropertyChip } from 'views-components/resource-properties-form/property-chip';
import { IllegalNamingWarning } from 'components/warning/warning';
import { GroupResource } from 'models/group';
import { UserResource } from 'models/user';
import { Link } from 'react-router-dom';
import { Link as ButtonLink } from '@mui/material';
import { ResourceWithName, ResponsiblePerson } from 'views-components/data-explorer/renderers';
import { MPVContainer, MPVPanelContent, MPVPanelState } from 'components/multi-panel-view/multi-panel-view';
import { resourceIsFrozen } from 'common/frozen-resources';
import { NotFoundView } from 'views/not-found-panel/not-found-panel';
import { setSelectedResourceUuid } from 'store/selected-resource/selected-resource-actions';

type CssRules =
    'root'
    | 'button'
    | 'infoCard'
    | 'propertiesCard'
    | 'filesCard'
    | 'iconHeader'
    | 'tag'
    | 'label'
    | 'value'
    | 'link'
    | 'centeredLabel'
    | 'warningLabel'
    | 'collectionName'
    | 'readOnlyIcon'
    | 'header'
    | 'title'
    | 'avatar'
    | 'content';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
    },
    button: {
        cursor: 'pointer'
    },
    infoCard: {
    },
    propertiesCard: {
        padding: 0,
    },
    filesCard: {
        padding: 0,
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.greyL
    },
    tag: {
        marginRight: theme.spacing(0.5),
        marginBottom: theme.spacing(0.5)
    },
    label: {
        fontSize: '0.875rem',
    },
    centeredLabel: {
        fontSize: '0.875rem',
        textAlign: 'center'
    },
    warningLabel: {
        fontStyle: 'italic'
    },
    collectionName: {
        flexDirection: 'column',
    },
    value: {
        textTransform: 'none',
        fontSize: '0.875rem'
    },
    link: {
        fontSize: '0.875rem',
        color: theme.palette.primary.main,
        '&:hover': {
            cursor: 'pointer'
        }
    },
    readOnlyIcon: {
        marginLeft: theme.spacing(1),
        fontSize: 'small',
    },
    header: {
        paddingTop: theme.spacing(1),
        paddingBottom: theme.spacing(1),
    },
    title: {
        overflow: 'hidden',
        paddingTop: theme.spacing(0.5),
        color: theme.customs.colors.green700,
    },
    avatar: {
        alignSelf: 'flex-start',
        paddingTop: theme.spacing(0.5),
    },
    content: {
        padding: theme.spacing(1),
        paddingTop: theme.spacing(0.5),
        '&:last-child': {
            paddingBottom: theme.spacing(1),
        }
    }
});

interface CollectionPanelDataProps {
    currentUserUUID: string;
    resources: ResourcesState;
}

type CollectionPanelProps = CollectionPanelDataProps & DispatchProp & WithStyles<CssRules>

type CollectionPanelState = {
    item: CollectionResource | null;
    itemOwner: GroupResource | UserResource | null;
    isWritable: boolean;
    isFrozen: boolean;
    isOldVersion: boolean
}

export const CollectionPanel = withStyles(styles)(connect(
    (state: RootState) => {
        return {
            currentUserUUID: state.auth.user?.uuid,
            resources: state.resources
        };
    })(
        class extends React.Component<CollectionPanelProps & RouteComponentProps<{ id: string }>> {
            state: CollectionPanelState = {
                item: null,
                itemOwner: null,
                isWritable: false,
                isFrozen: false,
                isOldVersion: false,
            }

            componentDidMount() {
                const item = this.props.resources[this.props.match.params.id] as CollectionResource;
                if (this.state.item) {
                    this.props.dispatch<any>(setSelectedResourceUuid(item.uuid))
                    this.setState({
                        item: item,
                        itemOwner: this.props.resources[item.ownerUuid] as GroupResource | UserResource,
                        isOldVersion: item.currentVersionUuid !== item.uuid,
                    });
                };
            }

            shouldComponentUpdate( nextProps: Readonly<CollectionPanelProps & RouteComponentProps<{ id: string }>>, nextState: Readonly<CollectionPanelState>, nextContext: any ): boolean {
                    return this.props.match.params.id !== nextProps.match.params.id
                        || this.props.resources !== nextProps.resources
                        || this.state.isWritable !== nextState.isWritable;
            }

            componentDidUpdate( prevProps: Readonly<CollectionPanelProps>, prevState: Readonly<CollectionPanelState>, snapshot?: any ): void {
                const { currentUserUUID, resources } = this.props;
                const item = this.props.resources[this.props.match.params.id] as CollectionResource;
                const itemOwner = this.props.resources[item.ownerUuid] as GroupResource | UserResource;
                if (item) {
                    if (prevState.item !== item) {
                        this.props.dispatch<any>(setSelectedResourceUuid(item.uuid))
                        this.setState({
                            item: item,
                            itemOwner: itemOwner,
                            isOldVersion: item.currentVersionUuid !== item.uuid,
                        });
                    }
                    if (prevProps.resources !== resources) {
                        const isWritable = this.checkIsWritable(item, itemOwner, currentUserUUID, resourceIsFrozen(item, resources));
                        this.setState({ isWritable: isWritable });
                    }
                }
            }

            checkIsWritable = (item: CollectionResource, itemOwner: GroupResource | UserResource | null, currentUserUUID: string, isFrozen: boolean): boolean => {
                let isWritable = false;

                if (item && !this.state.isOldVersion) {
                    if (item.ownerUuid === currentUserUUID) {
                        isWritable = true;
                    } else {
                        if (itemOwner) {
                            isWritable = itemOwner.canWrite;
                        }
                    }
                }
                if (item && isWritable) {
                    isWritable = !isFrozen;
                }
                return isWritable;
            }

            render() {
                const { classes, dispatch } = this.props;
                const { isWritable, item, isOldVersion } = this.state;
                const panelsData: MPVPanelState[] = [
                    { name: "Details" },
                    { name: "Files" },
                ];
                return item
                    ? <MPVContainer container className={classes.root} spacing={1} direction="column" justifyContent="flex-start" wrap="nowrap" panelStates={panelsData}>
                        <MPVPanelContent item xs="auto" data-cy='collection-info-panel'>
                            <Card className={classes.infoCard}>
                                <CardHeader
                                    className={classes.header}
                                    classes={{
                                        content: classes.title,
                                        avatar: classes.avatar,
                                    }}
                                    avatar={<IconButton onClick={this.openCollectionDetails} size="large">
                                        {isOldVersion
                                            ? <CollectionOldVersionIcon className={classes.iconHeader} />
                                            : <CollectionIcon className={classes.iconHeader} />}
                                    </IconButton>}
                                    title={
                                        <span>
                                            <IllegalNamingWarning name={item.name} />
                                            {item.name}
                                            {isWritable ||
                                                <Tooltip title="Read-only">
                                                    <span><ReadOnlyIcon data-cy="read-only-icon" className={classes.readOnlyIcon} /></span>
                                                </Tooltip>
                                                }
                                        </span>
                                    }
                                    action={
                                        <Tooltip title="Actions" disableFocusListener>
                                            <IconButton
                                                data-cy='collection-panel-options-btn'
                                                aria-label="Actions"
                                                onClick={this.handleContextMenu}
                                                size="large">
                                                <MoreVerticalIcon />
                                            </IconButton>
                                        </Tooltip>
                                    }
                                />
                                <CardContent className={classes.content}>
                                    <Typography variant="caption">
                                        {item.description}
                                    </Typography>
                                    <CollectionDetailsAttributes item={item} classes={classes} twoCol={true} showVersionBrowser={() => dispatch<any>(openDetailsPanel(item.uuid, 1))} />
                                    {(item.properties.container_request || item.properties.containerRequest) &&
                                        <span onClick={() => dispatch<any>(navigateToProcess(item.properties.container_request || item.properties.containerRequest))}>
                                            <DetailsAttribute classLabel={classes.link} label='Link to process' />
                                        </span>
                                    }
                                    {isOldVersion &&
                                        <Typography className={classes.warningLabel} variant="caption">
                                            This is an old version. Make a copy to make changes. Go to the <Link to={getCollectionUrl(item.currentVersionUuid)}>head version</Link> for sharing options.
                                        </Typography>
                                    }
                                </CardContent>
                            </Card>
                        </MPVPanelContent>
                        <MPVPanelContent item xs>
                            <Card className={classes.filesCard}>
                                <CollectionPanelFiles isWritable={isWritable} />
                            </Card>
                        </MPVPanelContent>
                    </MPVContainer >
                    : <NotFoundView
                        icon={CollectionIcon}
                        messages={["Collection not found"]}
                    />;
            }

            handleContextMenu = (event: React.MouseEvent<any>) => {
                if(this.state.item) {
                    const { uuid, ownerUuid, name, description,
                        kind, storageClassesDesired, properties } = this.state.item;
                    const menuKind = this.props.dispatch<any>(resourceUuidToContextMenuKind(uuid));
                    const resource = {
                        uuid,
                        ownerUuid,
                        name,
                        description,
                        storageClassesDesired,
                        kind,
                        menuKind,
                        properties,
                    };
                    // Avoid expanding/collapsing the panel
                    event.stopPropagation();
                    this.props.dispatch<any>(openContextMenu(event, resource));
                }
            }

            onCopy = (message: string) =>
                this.props.dispatch(snackbarActions.OPEN_SNACKBAR({
                    message,
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS
                }))

            openCollectionDetails = (e: React.MouseEvent<HTMLElement>) => {
                const { item } = this.state;
                if (item) {
                    e.stopPropagation();
                    this.props.dispatch<any>(openDetailsPanel(item.uuid));
                }
            }

            titleProps = {
                onClick: this.openCollectionDetails
            };

        }
    )
);

interface CollectionDetailsProps {
    item: CollectionResource;
    classes?: any;
    twoCol?: boolean;
    showVersionBrowser?: () => void;
}

export const CollectionDetailsAttributes = (props: CollectionDetailsProps) => {
    const item = props.item;
    const classes = props.classes || { label: '', value: '', button: '', tag: '' };
    const isOldVersion = item && item.currentVersionUuid !== item.uuid;
    const mdSize = props.twoCol ? 6 : 12;
    const showVersionBrowser = props.showVersionBrowser;
    const responsiblePersonRef = React.useRef(null);
    return <Grid container>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label={isOldVersion ? "This version's UUID" : "Collection UUID"}
                linkToUuid={item.uuid} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label={isOldVersion ? "This version's PDH" : "Portable data hash"}
                linkToUuid={item.portableDataHash} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Owner' linkToUuid={item.ownerUuid}
                uuidEnhancer={(uuid: string) => <ResourceWithName uuid={uuid} />} />
        </Grid>
        <div data-cy="responsible-person-wrapper" ref={responsiblePersonRef}>
            <Grid item xs={12} md={12}>
                <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                    label='Responsible person' linkToUuid={item.ownerUuid}
                    uuidEnhancer={(uuid: string) => <ResponsiblePerson uuid={item.uuid} parentRef={responsiblePersonRef.current} />} />
            </Grid>
        </div>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Head version'
                value={isOldVersion ? undefined : 'this one'}
                linkToUuid={isOldVersion ? item.currentVersionUuid : undefined} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute
                classLabel={classes.label} classValue={classes.value}
                label='Version number'
                value={showVersionBrowser !== undefined
                    ? <Tooltip title="Open version browser"><ButtonLink underline='none' className={classes.button} onClick={() => showVersionBrowser()}>
                        {<span data-cy='collection-version-number'>{item.version}</span>}
                    </ButtonLink></Tooltip>
                    : item.version
                }
            />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute label='Created at' value={formatDate(item.createdAt)} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute label='Last modified' value={formatDate(item.modifiedAt)} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Number of files' value={<span data-cy='collection-file-count'>{item.fileCount}</span>} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Content size' value={formatFileSize(item.fileSizeTotal)} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Storage classes' value={item.storageClassesDesired ? item.storageClassesDesired.join(', ') : ["default"]} />
        </Grid>

        {/*
            NOTE: The property list should be kept at the bottom, because it spans
            the entire available width, without regards of the twoCol prop.
          */}
        <Grid item xs={12} md={12}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Properties' />
            {item.properties && Object.keys(item.properties).length > 0
                ? Object.keys(item.properties).map(k =>
                    Array.isArray(item.properties[k])
                        ? item.properties[k].map((v: string) =>
                            getPropertyChip(k, v, undefined, classes.tag))
                        : getPropertyChip(k, item.properties[k], undefined, classes.tag))
                : <div className={classes.value}>No properties</div>}
        </Grid>
    </Grid>;
};
