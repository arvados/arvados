// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import {
    StyleRulesCallback,
    WithStyles,
    withStyles,
    IconButton,
    Grid,
    Tooltip,
    Typography,
    Card, CardHeader, CardContent
} from '@material-ui/core';
import { connect, DispatchProp } from "react-redux";
import { RouteComponentProps } from 'react-router';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { MoreVerticalIcon, CollectionIcon, ReadOnlyIcon, CollectionOldVersionIcon } from 'components/icon/icon';
import { DetailsAttribute } from 'components/details-attribute/details-attribute';
import { CollectionResource, getCollectionUrl } from 'models/collection';
import { CollectionPanelFiles } from 'views-components/collection-panel-files/collection-panel-files';
import { navigateToProcess } from 'store/collection-panel/collection-panel-action';
import { getResource } from 'store/resources/resources';
import { openContextMenu, resourceUuidToContextMenuKind } from 'store/context-menu/context-menu-actions';
import { formatDate, formatFileSize } from "common/formatters";
import { openDetailsPanel } from 'store/details-panel/details-panel-action';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { getPropertyChip } from 'views-components/resource-properties-form/property-chip';
import { IllegalNamingWarning } from 'components/warning/warning';
import { GroupResource } from 'models/group';
import { UserResource } from 'models/user';
import { getUserUuid } from 'common/getuser';
import { Link } from 'react-router-dom';
import { Link as ButtonLink } from '@material-ui/core';
import { ResourceWithName, ResponsiblePerson } from 'views-components/data-explorer/renderers';
import { MPVContainer, MPVPanelContent, MPVPanelState } from 'components/multi-panel-view/multi-panel-view';
import { resourceIsFrozen } from 'common/frozen-resources';
import { NotFoundView } from 'views/not-found-panel/not-found-panel';

type CssRules = 'root'
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

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
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
        marginRight: theme.spacing.unit / 2,
        marginBottom: theme.spacing.unit / 2
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
        marginLeft: theme.spacing.unit,
        fontSize: 'small',
    },
    header: {
        paddingTop: theme.spacing.unit,
        paddingBottom: theme.spacing.unit,
    },
    title: {
        overflow: 'hidden',
        paddingTop: theme.spacing.unit * 0.5,
        color: theme.customs.colors.green700,
    },
    avatar: {
        alignSelf: 'flex-start',
        paddingTop: theme.spacing.unit * 0.5
    },
    content: {
        padding: theme.spacing.unit * 1.0,
        paddingTop: theme.spacing.unit * 0.5,
        '&:last-child': {
            paddingBottom: theme.spacing.unit * 1,
        }
    }
});

interface CollectionPanelDataProps {
    item: CollectionResource;
    isWritable: boolean;
    isOldVersion: boolean;
    isLoadingFiles: boolean;
}

type CollectionPanelProps = CollectionPanelDataProps & DispatchProp & WithStyles<CssRules>

export const CollectionPanel = withStyles(styles)(connect(
    (state: RootState, props: RouteComponentProps<{ id: string }>) => {
        const currentUserUUID = getUserUuid(state);
        const item = getResource<CollectionResource>(props.match.params.id)(state.resources);
        let isWritable = false;
        const isOldVersion = item && item.currentVersionUuid !== item.uuid;
        if (item && !isOldVersion) {
            if (item.ownerUuid === currentUserUUID) {
                isWritable = true;
            } else {
                const itemOwner = getResource<GroupResource | UserResource>(item.ownerUuid)(state.resources);
                if (itemOwner) {
                    isWritable = itemOwner.canWrite;
                }
            }
        }

        if (item && isWritable) {
            isWritable = !resourceIsFrozen(item, state.resources);
        }

        return { item, isWritable, isOldVersion };
    })(
        class extends React.Component<CollectionPanelProps> {
            render() {
                const { classes, item, dispatch, isWritable, isOldVersion } = this.props;
                const panelsData: MPVPanelState[] = [
                    { name: "Details" },
                    { name: "Files" },
                ];
                return item
                    ? <MPVContainer className={classes.root} spacing={8} direction="column" justify-content="flex-start" wrap="nowrap" panelStates={panelsData}>
                        <MPVPanelContent xs="auto" data-cy='collection-info-panel'>
                            <Card className={classes.infoCard}>
                                <CardHeader
                                    className={classes.header}
                                    classes={{
                                        content: classes.title,
                                        avatar: classes.avatar,
                                    }}
                                    avatar={<IconButton onClick={this.openCollectionDetails}>
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
                                                    <ReadOnlyIcon data-cy="read-only-icon" className={classes.readOnlyIcon} />
                                                </Tooltip>}
                                        </span>
                                    }
                                    action={
                                        <Tooltip title="Actions" disableFocusListener>
                                            <IconButton
                                                data-cy='collection-panel-options-btn'
                                                aria-label="Actions"
                                                onClick={this.handleContextMenu}>
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
                        <MPVPanelContent xs>
                            <Card className={classes.filesCard}>
                                <CollectionPanelFiles isWritable={isWritable} />
                            </Card>
                        </MPVPanelContent>
                    </MPVContainer >
                    : <NotFoundView
                        icon={CollectionIcon}
                        messages={["Collection not found"]}
                    />
                    ;
            }

            handleContextMenu = (event: React.MouseEvent<any>) => {
                const { uuid, ownerUuid, name, description,
                    kind, storageClassesDesired, properties } = this.props.item;
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

            onCopy = (message: string) =>
                this.props.dispatch(snackbarActions.OPEN_SNACKBAR({
                    message,
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS
                }))

            openCollectionDetails = (e: React.MouseEvent<HTMLElement>) => {
                const { item } = this.props;
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
                label='Storage classes' value={item.storageClassesDesired.join(', ')} />
        </Grid>

        {/*
            NOTE: The property list should be kept at the bottom, because it spans
            the entire available width, without regards of the twoCol prop.
          */}
        <Grid item xs={12} md={12}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Properties' />
            {Object.keys(item.properties).length > 0
                ? Object.keys(item.properties).map(k =>
                    Array.isArray(item.properties[k])
                        ? item.properties[k].map((v: string) =>
                            getPropertyChip(k, v, undefined, classes.tag))
                        : getPropertyChip(k, item.properties[k], undefined, classes.tag))
                : <div className={classes.value}>No properties</div>}
        </Grid>
    </Grid>;
};
