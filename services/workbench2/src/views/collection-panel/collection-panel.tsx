// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Grid, Tooltip, Typography, CardContent } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { connect, DispatchProp } from "react-redux";
import { RouteComponentProps } from 'react-router';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { CollectionIcon } from 'components/icon/icon';
import { DetailsAttribute } from 'components/details-attribute/details-attribute';
import { CollectionResource, getCollectionUrl } from 'models/collection';
import { CollectionPanelFiles } from 'views-components/collection-panel-files/collection-panel-files';
import { navigateToProcess } from 'store/collection-panel/collection-panel-action';
import { ResourcesState, getResource } from 'store/resources/resources';
import { openContextMenuAndSelect } from 'store/context-menu/context-menu-actions';
import { formatDate, formatFileSize } from "common/formatters";
import { openDetailsPanel } from 'store/details-panel/details-panel-action';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { getPropertyChip } from 'views-components/resource-properties-form/property-chip';
import { GroupResource } from 'models/group';
import { UserResource } from 'models/user';
import { Link } from 'react-router-dom';
import { Link as ButtonLink } from '@mui/material';
import { ResourceWithName, ResponsiblePerson } from 'views-components/data-explorer/renderers';
import { MPVContainer, MPVPanelContent, MPVPanelState } from 'components/multi-panel-view/multi-panel-view';
import { resourceIsFrozen } from 'common/frozen-resources';
import { NotFoundView } from 'views/not-found-panel/not-found-panel';
import { setSelectedResourceUuid } from 'store/selected-resource/selected-resource-actions';
import { resourceToMenuKind } from 'common/resource-to-menu-kind';
import { collectionPanelActions } from 'store/collection-panel/collection-panel-action';
import { DetailsCardRoot } from 'views-components/details-card/details-card-root';
import { CollapsibleDescription } from 'components/collapsible-description/collapsible-description';
import { ExpandChevronRight } from 'components/expand-chevron-right/expand-chevron-right';

type CssRules =
    'root'
    | 'mpvRoot'
    | 'button'
    | 'infoCard'
    | 'filesCard'
    | 'tag'
    | 'label'
    | 'value'
    | 'link'
    | 'warningLabel'
    | 'content';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        display: 'flex',
        flexDirection: 'column',
    },
    mpvRoot: {
        width: '100%',
        height: '100%',
    },
    button: {
        cursor: 'pointer'
    },
    infoCard: {
    },
    filesCard: {
        padding: 0,
    },
    tag: {
        marginRight: theme.spacing(0.5),
        marginBottom: theme.spacing(0.5)
    },
    label: {
        fontSize: '0.875rem',
    },
    warningLabel: {
        fontStyle: 'italic'
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
    isOldVersion: boolean;
    hasDescription: boolean;
    showDescription: boolean;
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
                isOldVersion: false,
                hasDescription: false,
                showDescription: false,
            }

            shouldComponentUpdate( nextProps: Readonly<CollectionPanelProps & RouteComponentProps<{ id: string }>>, nextState: Readonly<CollectionPanelState>, nextContext: any ): boolean {
                    return this.props.match.params.id !== nextProps.match.params.id
                        || this.props.resources !== nextProps.resources
                        || this.state.isWritable !== nextState.isWritable
                        || this.state.hasDescription !== nextState.hasDescription
                        || this.state.showDescription !== nextState.showDescription;
            }

            componentDidUpdate( prevProps: Readonly<CollectionPanelProps>, prevState: Readonly<CollectionPanelState>, snapshot?: any ): void {
                const { currentUserUUID, resources } = this.props;
                const collection = getResource<CollectionResource>(this.props.match.params.id)(this.props.resources);
                if (!this.state.item && collection) this.setState({ item: collection });
                if (collection) {
                    this.setState({
                        hasDescription: collection.description && collection.description.length > 0,
                    });
                    const itemOwner = collection ? getResource<GroupResource | UserResource>(collection.ownerUuid)(this.props.resources) : undefined;
                    if (prevState.item !== collection) {
                        this.props.dispatch<any>(setSelectedResourceUuid(collection.uuid))
                        this.setState({
                            item: collection,
                            itemOwner: itemOwner,
                            isOldVersion: collection.currentVersionUuid !== collection.uuid,
                        });
                    }
                    if (prevProps.resources !== resources && itemOwner) {
                        const isWritable = this.checkIsWritable(collection, itemOwner, currentUserUUID, resourceIsFrozen(collection, resources));
                        this.setState({ isWritable: isWritable });
                    }
                }
            }

            componentWillUnmount(): void {
                this.props.dispatch<any>(collectionPanelActions.RESET_COLLECTION_PANEL());
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

            setShowDescription = (showDescription: boolean) => {
                this.setState({ showDescription });
            }

            render() {
                const { classes, dispatch } = this.props;
                const { isWritable, item, isOldVersion, hasDescription, showDescription } = this.state;
                const panelsData: MPVPanelState[] = [
                    { name: "Overview" },
                    { name: "Files" },
                ];
                return item
                    ? <section className={classes.root}>
                        <DetailsCardRoot />
                        <MPVContainer container className={classes.mpvRoot} justifyContent="flex-start" panelStates={panelsData}>
                            <MPVPanelContent item xs="auto" data-cy='collection-info-panel'>
                                <section className={classes.infoCard}>
                                    <CardContent className={classes.content}>
                                        <Grid item xs={12} md={12} onClick={() => this.setShowDescription(!showDescription)}>
                                            <DetailsAttribute label={'Description'} button={hasDescription ? <ExpandChevronRight expanded={showDescription} /> : undefined}>
                                                {hasDescription
                                                    ? <CollapsibleDescription description={item.description} showDescription={showDescription} />
                                                    : <Typography>No description available</Typography>}
                                            </DetailsAttribute>
                                        </Grid>
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
                                </section>
                            </MPVPanelContent>
                            <MPVPanelContent item xs>
                                <section className={classes.filesCard}>
                                    <CollectionPanelFiles isWritable={isWritable} />
                                </section>
                            </MPVPanelContent>
                        </MPVContainer >
                    </section>
                    : <NotFoundView
                        icon={CollectionIcon}
                        messages={["Collection not found"]}
                    />;
            }

            handleContextMenu = (event: React.MouseEvent<any>) => {
                if (!this.state.item) return;
                const { uuid, ownerUuid, name, description,
                    kind, storageClassesDesired, properties } = this.state.item;
                const menuKind = this.props.dispatch<any>(resourceToMenuKind(uuid));
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
                this.props.dispatch<any>(openContextMenuAndSelect(event, resource));
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
