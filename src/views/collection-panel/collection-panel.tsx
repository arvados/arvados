// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    StyleRulesCallback, WithStyles, withStyles, Card,
    CardHeader, IconButton, CardContent, Grid, Tooltip
} from '@material-ui/core';
import MuiExpansionPanel from '@material-ui/core/ExpansionPanel';
import MuiExpansionPanelSummary from '@material-ui/core/ExpansionPanelSummary';
import MuiExpansionPanelDetails from '@material-ui/core/ExpansionPanelDetails';
import { connect, DispatchProp } from "react-redux";
import { RouteComponentProps } from 'react-router';
import { ArvadosTheme } from '~/common/custom-theme';
import { RootState } from '~/store/store';
import { MoreOptionsIcon, CollectionIcon, ReadOnlyIcon, ExpandIcon } from '~/components/icon/icon';
import { DetailsAttribute } from '~/components/details-attribute/details-attribute';
import { CollectionResource } from '~/models/collection';
import { CollectionPanelFiles } from '~/views-components/collection-panel-files/collection-panel-files';
import { CollectionTagForm } from './collection-tag-form';
import { deleteCollectionTag, navigateToProcess, collectionPanelActions } from '~/store/collection-panel/collection-panel-action';
import { getResource } from '~/store/resources/resources';
import { openContextMenu } from '~/store/context-menu/context-menu-actions';
import { ContextMenuKind } from '~/views-components/context-menu/context-menu';
import { formatFileSize } from "~/common/formatters";
import { openDetailsPanel } from '~/store/details-panel/details-panel-action';
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { getPropertyChip } from '~/views-components/resource-properties-form/property-chip';
import { IllegalNamingWarning } from '~/components/warning/warning';
import { GroupResource } from '~/models/group';
import { UserResource } from '~/models/user';
import { getUserUuid } from '~/common/getuser';
import { getProgressIndicator } from '~/store/progress-indicator/progress-indicator-reducer';
import { COLLECTION_PANEL_LOAD_FILES, loadCollectionFiles, COLLECTION_PANEL_LOAD_FILES_THRESHOLD } from '~/store/collection-panel/collection-panel-files/collection-panel-files-actions';

type CssRules = 'root'
    | 'card'
    | 'cardHeader'
    | 'filesCard'
    | 'cardContent'
    | 'iconHeader'
    | 'tag'
    | 'label'
    | 'value'
    | 'link'
    | 'centeredLabel'
    | 'readOnlyIcon';

const ExpansionPanel = withStyles({
    root: {
        margin: 0,
        padding: 0,
    }
})(MuiExpansionPanel);

const ExpansionPanelSummary = withStyles({
    root: {
        margin: 0,
        padding: 0,
    },
    content: {
        '&$expanded': {
            margin: 0,
            padding: 0,
        },
        margin: 0,
        padding: 0,
    },
    expanded: {},
})(MuiExpansionPanelSummary);

const ExpansionPanelDetails = withStyles({
    root: {
        margin: 0,
        padding: 0,
    }
})(MuiExpansionPanelDetails);

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        display: 'flex',
        flexFlow: 'column',
        height: 'calc(100vh - 130px)', // (100% viewport height) - (top bar + breadcrumbs)
    },
    card: {
        marginBottom: theme.spacing.unit * 2,
    },
    cardHeader: {
        paddingTop: theme.spacing.unit,
        paddingBottom: 0,
    },
    filesCard: {
        marginBottom: theme.spacing.unit * 2,
        flex: 1,
    },
    cardContent: {
        width: '100%',
        paddingTop: 0,
        paddingBottom: theme.spacing.unit,
        '&:last-child': {
            paddingBottom: theme.spacing.unit,
        }
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.yellow700
    },
    tag: {
        marginRight: theme.spacing.unit,
        marginBottom: theme.spacing.unit
    },
    label: {
        fontSize: '0.875rem'
    },
    centeredLabel: {
        fontSize: '0.875rem',
        textAlign: 'center'
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
    }
});

interface CollectionPanelDataProps {
    item: CollectionResource;
    isWritable: boolean;
    isLoadingFiles: boolean;
    tooManyFiles: boolean;
}

type CollectionPanelProps = CollectionPanelDataProps & DispatchProp
    & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;

export const CollectionPanel = withStyles(styles)(
    connect((state: RootState, props: RouteComponentProps<{ id: string }>) => {
        const currentUserUUID = getUserUuid(state);
        const item = getResource<CollectionResource>(props.match.params.id)(state.resources);
        let isWritable = false;
        if (item && item.ownerUuid === currentUserUUID) {
            isWritable = true;
        } else if (item) {
            const itemOwner = getResource<GroupResource|UserResource>(item.ownerUuid)(state.resources);
            if (itemOwner) {
                isWritable = itemOwner.writableBy.indexOf(currentUserUUID || '') >= 0;
            }
        }
        const loadingFilesIndicator = getProgressIndicator(COLLECTION_PANEL_LOAD_FILES)(state.progressIndicator);
        const isLoadingFiles = loadingFilesIndicator && loadingFilesIndicator!.working || false;
        const tooManyFiles = !state.collectionPanel.loadBigCollections && item && item.fileCount > COLLECTION_PANEL_LOAD_FILES_THRESHOLD || false;
        return { item, isWritable, isLoadingFiles, tooManyFiles };
    })(
        class extends React.Component<CollectionPanelProps> {
            render() {
                const { classes, item, dispatch, isWritable, isLoadingFiles, tooManyFiles } = this.props;
                return item
                    ? <div className={classes.root}>
                        <Card data-cy='collection-info-panel' className={classes.card}>
                        <ExpansionPanel defaultExpanded>
                        <ExpansionPanelSummary expandIcon={<ExpandIcon />}>
                            <CardHeader
                                className={classes.cardHeader}
                                avatar={
                                    <IconButton onClick={this.openCollectionDetails}>
                                        <CollectionIcon className={classes.iconHeader} />
                                    </IconButton>
                                }
                                title={
                                    <span>
                                        <IllegalNamingWarning name={item.name}/>
                                        {item.name}
                                        {isWritable ||
                                        <Tooltip title="Read-only">
                                            <ReadOnlyIcon data-cy="read-only-icon" className={classes.readOnlyIcon} />
                                        </Tooltip>
                                        }
                                    </span>
                                }
                                titleTypographyProps={this.titleProps}
                                subheader={item.description}
                                subheaderTypographyProps={this.titleProps} />
                            </ExpansionPanelSummary>
                            <ExpansionPanelDetails>
                            <CardContent className={classes.cardContent}>
                                <Grid container justify="space-between">
                                    <Grid item xs={11}>
                                        <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                            label='Collection UUID'
                                            linkToUuid={item.uuid} />
                                        <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                            label='Portable data hash'
                                            linkToUuid={item.portableDataHash} />
                                        <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                            label='Number of files' value={item.fileCount} />
                                        <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                            label='Content size' value={formatFileSize(item.fileSizeTotal)} />
                                        <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                            label='Owner' linkToUuid={item.ownerUuid} />
                                        {(item.properties.container_request || item.properties.containerRequest) &&
                                            <span onClick={() => dispatch<any>(navigateToProcess(item.properties.container_request || item.properties.containerRequest))}>
                                                <DetailsAttribute classLabel={classes.link} label='Link to process' />
                                            </span>
                                        }
                                    </Grid>
                                    <Grid item xs={1} style={{textAlign: "right"}}>
                                        <Tooltip title="More options" disableFocusListener>
                                            <IconButton
                                                data-cy='collection-panel-options-btn'
                                                aria-label="More options"
                                                onClick={this.handleContextMenu}>
                                                <MoreOptionsIcon />
                                            </IconButton>
                                        </Tooltip>
                                    </Grid>
                                </Grid>
                            </CardContent>
                            </ExpansionPanelDetails>
                        </ExpansionPanel>
                        </Card>

                        <Card data-cy='collection-properties-panel' className={classes.card}>
                        <ExpansionPanel defaultExpanded>
                            <ExpansionPanelSummary expandIcon={<ExpandIcon />}>
                                <CardHeader title="Properties" />
                            </ExpansionPanelSummary>
                            <ExpansionPanelDetails><CardContent className={classes.cardContent}>
                                <Grid container direction="column">
                                    {isWritable && <Grid item xs={12}>
                                        <CollectionTagForm />
                                    </Grid>}
                                    <Grid item xs={12}>
                                    { Object.keys(item.properties).length > 0
                                        ? Object.keys(item.properties).map(k =>
                                            Array.isArray(item.properties[k])
                                            ? item.properties[k].map((v: string) =>
                                                getPropertyChip(
                                                    k, v,
                                                    isWritable
                                                        ? this.handleDelete(k, item.properties[k])
                                                        : undefined,
                                                    classes.tag))
                                            : getPropertyChip(
                                                k, item.properties[k],
                                                isWritable
                                                    ? this.handleDelete(k, item.properties[k])
                                                    : undefined,
                                                classes.tag)
                                        )
                                        : <div className={classes.centeredLabel}>No properties set on this collection.</div>
                                    }
                                    </Grid>
                                </Grid>
                            </CardContent></ExpansionPanelDetails>
                        </ExpansionPanel>
                        </Card>
                        <div className={classes.filesCard}>
                            <CollectionPanelFiles
                                isWritable={isWritable}
                                isLoading={isLoadingFiles}
                                tooManyFiles={tooManyFiles}
                                loadFilesFunc={() => {
                                    dispatch(collectionPanelActions.LOAD_BIG_COLLECTIONS(true));
                                    dispatch<any>(loadCollectionFiles(this.props.item.uuid));
                                }
                            } />
                        </div>
                    </div>
                    : null;
            }

            handleContextMenu = (event: React.MouseEvent<any>) => {
                const { uuid, ownerUuid, name, description, kind, isTrashed } = this.props.item;
                const { isWritable } = this.props;
                const resource = {
                    uuid,
                    ownerUuid,
                    name,
                    description,
                    kind,
                    menuKind: isWritable
                        ? isTrashed
                            ? ContextMenuKind.TRASHED_COLLECTION
                            : ContextMenuKind.COLLECTION
                        : ContextMenuKind.READONLY_COLLECTION
                };
                this.props.dispatch<any>(openContextMenu(event, resource));
            }

            onCopy = (message: string) =>
                this.props.dispatch(snackbarActions.OPEN_SNACKBAR({
                    message,
                    hideDuration: 2000,
                    kind: SnackbarKind.SUCCESS
                }))

            handleDelete = (key: string, value: string) => () => {
                this.props.dispatch<any>(deleteCollectionTag(key, value));
            }

            openCollectionDetails = () => {
                const { item } = this.props;
                if (item) {
                    this.props.dispatch(openDetailsPanel(item.uuid));
                }
            }

            titleProps = {
                onClick: this.openCollectionDetails
            };

        }
    )
);
