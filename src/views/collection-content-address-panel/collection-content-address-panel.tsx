// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Link } from 'react-router-dom';
import {
    StyleRulesCallback, WithStyles, withStyles, Grid
} from '@material-ui/core';
import { CollectionIcon } from '~/components/icon/icon';
import { ArvadosTheme } from '~/common/custom-theme';
import { BackIcon } from '~/components/icon/icon';
import { CollectionResource } from '~/models/collection';
import { DataTableDefaultView } from '~/components/data-table-default-view/data-table-default-view';
import { COLLECTIONS_CONTENT_ADDRESS_PANEL_ID } from '~/store/collections-content-address-panel/collections-content-address-panel-actions';
import { DataExplorer } from "~/views-components/data-explorer/data-explorer";
import { Dispatch } from 'redux';
import { getIsAdmin } from '~/store/public-favorites/public-favorites-actions';
import { resourceKindToContextMenuKind, openContextMenu } from '~/store/context-menu/context-menu-actions';
import { ResourceKind } from '~/models/resource';
import { loadDetailsPanel } from '~/store/details-panel/details-panel-action';
import { connect, DispatchProp } from 'react-redux';
import { navigateTo } from '~/store/navigation/navigation-action';

type CssRules = 'backLink' | 'backIcon' | 'card' | 'title' | 'iconHeader' | 'link';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    backLink: {
        fontSize: '1rem',
        fontWeight: 600,
        display: 'flex',
        alignItems: 'center',
        textDecoration: 'none',
        padding: theme.spacing.unit,
        color: theme.palette.grey["700"],
    },
    backIcon: {
        marginRight: theme.spacing.unit
    },
    card: {
        width: '100%'
    },
    title: {
        color: theme.palette.grey["700"]
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.green700
    },
    link: {
        fontSize: '0.875rem',
        color: theme.palette.primary.main,
        textAlign: 'right',
        '&:hover': {
            cursor: 'pointer'
        }
    }
});


export interface CollectionContentAddressMainCardActionProps {
    onContextMenu: (event: React.MouseEvent<any>, uuid: string) => void;
    onItemClick: (item: string) => void;
    onItemDoubleClick: (item: string) => void;
}

const mapDispatchToProps = (dispatch: Dispatch): CollectionContentAddressMainCardActionProps => ({
    onContextMenu: (event, resourceUuid) => {
        const isAdmin = dispatch<any>(getIsAdmin());
        const kind = resourceKindToContextMenuKind(resourceUuid, isAdmin);
        if (kind) {
            dispatch<any>(openContextMenu(event, {
                name: '',
                uuid: resourceUuid,
                ownerUuid: '',
                kind: ResourceKind.NONE,
                menuKind: kind
            }));
        }
        dispatch<any>(loadDetailsPanel(resourceUuid));
    },
    onItemClick: (uuid: string) => {
        dispatch<any>(loadDetailsPanel(uuid));
    },
    onItemDoubleClick: uuid => {
        dispatch<any>(navigateTo(uuid));
    }
});

export const CollectionsContentAddressPanel = withStyles(styles)(
    connect(null, mapDispatchToProps)(
        class extends React.Component<CollectionContentAddressMainCardActionProps & WithStyles<CssRules>> {
            render() {
                return <Grid item xs={12}>
                    {/* <Link to={`/collections/${this.props.collection.uuid}`} className={this.props.classes.backLink}>
                        <BackIcon className={this.props.classes.backIcon} />
                        Back test
                    </Link> */}
                    <DataExplorer
                        id={COLLECTIONS_CONTENT_ADDRESS_PANEL_ID}
                        onRowClick={this.props.onItemClick}
                        onRowDoubleClick={this.props.onItemDoubleClick}
                        onContextMenu={this.props.onContextMenu}
                        contextMenuColumn={true}
                        dataTableDefaultView={
                            <DataTableDefaultView
                                icon={CollectionIcon}
                                messages={['Collections with this content address not found.']} />
                        } />;
                    </Grid >;
            }
        }
    )
);