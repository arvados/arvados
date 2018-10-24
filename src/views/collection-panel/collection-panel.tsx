// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    StyleRulesCallback, WithStyles, withStyles, Card,
    CardHeader, IconButton, CardContent, Grid, Chip, Tooltip
} from '@material-ui/core';
import { connect, DispatchProp } from "react-redux";
import { RouteComponentProps } from 'react-router';
import { ArvadosTheme } from '~/common/custom-theme';
import { RootState } from '~/store/store';
import { MoreOptionsIcon, CollectionIcon, CopyIcon } from '~/components/icon/icon';
import { DetailsAttribute } from '~/components/details-attribute/details-attribute';
import { CollectionResource } from '~/models/collection';
import { CollectionPanelFiles } from '~/views-components/collection-panel-files/collection-panel-files';
import * as CopyToClipboard from 'react-copy-to-clipboard';
import { CollectionTagForm } from './collection-tag-form';
import { deleteCollectionTag } from '~/store/collection-panel/collection-panel-action';
import { snackbarActions } from '~/store/snackbar/snackbar-actions';
import { getResource } from '~/store/resources/resources';
import { openContextMenu } from '~/store/context-menu/context-menu-actions';
import { ContextMenuKind } from '~/views-components/context-menu/context-menu';
import { formatFileSize } from "~/common/formatters";
import { getResourceData } from "~/store/resources-data/resources-data";
import { ResourceData } from "~/store/resources-data/resources-data-reducer";

type CssRules = 'card' | 'iconHeader' | 'tag' | 'copyIcon' | 'label' | 'value';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        marginBottom: theme.spacing.unit * 2
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.yellow700
    },
    tag: {
        marginRight: theme.spacing.unit,
        marginBottom: theme.spacing.unit
    },
    copyIcon: {
        marginLeft: theme.spacing.unit,
        fontSize: '1.125rem',
        color: theme.palette.grey["500"],
        cursor: 'pointer'
    },
    label: {
        fontSize: '0.875rem'
    },
    value: {
        textTransform: 'none',
        fontSize: '0.875rem'
    }
});

interface CollectionPanelDataProps {
    item: CollectionResource;
    data: ResourceData;
}

type CollectionPanelProps = CollectionPanelDataProps & DispatchProp
    & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;


export const CollectionPanel = withStyles(styles)(
    connect((state: RootState, props: RouteComponentProps<{ id: string }>) => {
        const item = getResource(props.match.params.id)(state.resources);
        const data = getResourceData(props.match.params.id)(state.resourcesData);
        return { item, data };
    })(
        class extends React.Component<CollectionPanelProps> {
            render() {
                const { classes, item, data } = this.props;
                return item
                    ? <>
                        <Card className={classes.card}>
                            <CardHeader
                                avatar={<CollectionIcon className={classes.iconHeader} />}
                                action={
                                    <Tooltip title="More options">
                                        <IconButton
                                            aria-label="More options"
                                            onClick={this.handleContextMenu}>
                                            <MoreOptionsIcon />
                                        </IconButton>
                                    </Tooltip>
                                }
                                title={item && item.name}
                                subheader={item && item.description} />
                            <CardContent>
                                <Grid container direction="column">
                                    <Grid item xs={6}>
                                        <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                            label='Collection UUID'
                                            value={item && item.uuid}>
                                            <Tooltip title="Copy uuid">
                                                <CopyToClipboard text={item && item.uuid} onCopy={() => this.onCopy()}>
                                                    <CopyIcon className={classes.copyIcon} />
                                                </CopyToClipboard>
                                            </Tooltip>
                                        </DetailsAttribute>
                                        <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                            label='Number of files' value={data && data.fileCount} />
                                        <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                            label='Content size' value={data && formatFileSize(data.fileSize)} />
                                        <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                            label='Owner' value={item && item.ownerUuid} />
                                    </Grid>
                                </Grid>
                            </CardContent>
                        </Card>

                        <Card className={classes.card}>
                            <CardHeader title="Properties" />
                            <CardContent>
                                <Grid container direction="column">
                                    <Grid item xs={12}>
                                        <CollectionTagForm />
                                    </Grid>
                                    <Grid item xs={12}>
                                        {
                                            Object.keys(item.properties).map(k => {
                                                return <Chip key={k} className={classes.tag}
                                                    onDelete={this.handleDelete(k)}
                                                    label={`${k}: ${item.properties[k]}`} />;
                                            })
                                        }
                                    </Grid>
                                </Grid>
                            </CardContent>
                        </Card>
                        <div className={classes.card}>
                            <CollectionPanelFiles />
                        </div>
                    </>
                    : null;
            }

            handleContextMenu = (event: React.MouseEvent<any>) => {
                const { uuid, ownerUuid, name, description, kind, isTrashed } = this.props.item;
                const resource = {
                    uuid,
                    ownerUuid,
                    name,
                    description,
                    kind,
                    menuKind: isTrashed
                        ? ContextMenuKind.TRASHED_COLLECTION
                        : ContextMenuKind.COLLECTION
                };
                this.props.dispatch<any>(openContextMenu(event, resource));
            }

            handleDelete = (key: string) => () => {
                this.props.dispatch<any>(deleteCollectionTag(key));
            }

            onCopy = () => {
                this.props.dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: "Uuid has been copied",
                    hideDuration: 2000
                }));
            }
        }
    )
);
