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
import { TagResource } from '~/models/tag';
import { CollectionTagForm } from './collection-tag-form';
import { deleteCollectionTag } from '~/store/collection-panel/collection-panel-action';
import { snackbarActions } from '~/store/snackbar/snackbar-actions';
import { getResource } from '~/store/resources/resources';
import { contextMenuActions, openContextMenu } from '~/store/context-menu/context-menu-actions';
import { ContextMenuKind } from '~/views-components/context-menu/context-menu';

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
    tags: TagResource[];
}

type CollectionPanelProps = CollectionPanelDataProps & DispatchProp
    & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;


export const CollectionPanel = withStyles(styles)(
    connect((state: RootState, props: RouteComponentProps<{ id: string }>) => {
        const collection = getResource(props.match.params.id)(state.resources);
        return {
            item: collection,
            tags: state.collectionPanel.tags
        };
    })(
        class extends React.Component<CollectionPanelProps> {

            render() {
                const { classes, item, tags } = this.props;
                return <div>
                    <Card className={classes.card}>
                        <CardHeader
                            avatar={<CollectionIcon className={classes.iconHeader} />}
                            action={
                                <IconButton
                                    aria-label="More options"
                                    onClick={this.handleContextMenu}>
                                    <MoreOptionsIcon />
                                </IconButton>
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
                                        label='Number of files' value='14' />
                                    <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                        label='Content size' value='54 MB' />
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
                                <Grid item xs={12}><CollectionTagForm /></Grid>
                                <Grid item xs={12}>
                                    {
                                        tags.map(tag => {
                                            return <Chip key={tag.etag} className={classes.tag}
                                                onDelete={this.handleDelete(tag.uuid)}
                                                label={renderTagLabel(tag)} />;
                                        })
                                    }
                                </Grid>
                            </Grid>
                        </CardContent>
                    </Card>
                    <div className={classes.card}>
                        <CollectionPanelFiles />
                    </div>
                </div>;
            }

            handleContextMenu = (event: React.MouseEvent<any>) => {
                const { uuid, name, description } = this.props.item;
                const resource = {
                    uuid,
                    name,
                    description,
                    kind: ContextMenuKind.COLLECTION
                };
                this.props.dispatch<any>(openContextMenu(event, resource));
            }

            handleDelete = (uuid: string) => () => {
                this.props.dispatch<any>(deleteCollectionTag(uuid));
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

const renderTagLabel = (tag: TagResource) => {
    const { properties } = tag;
    return `${properties.key}: ${properties.value}`;
};
