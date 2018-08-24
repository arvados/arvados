// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { Breadcrumbs as BreadcrumbsComponent, BreadcrumbsProps } from '~/components/breadcrumbs/breadcrumbs';
import { RootState } from '~/store/store';
import { Breadcrumb } from '~/components/breadcrumbs/breadcrumbs';
import { matchProjectRoute } from '~/routes/routes';
import { getTreePicker } from '~/store/tree-picker/tree-picker';
import { SIDE_PANEL_TREE } from '~/store/side-panel-tree/side-panel-tree-actions';
import { getNodeAncestors, getNode } from '~/models/tree';
import { Dispatch } from 'redux';
import { navigateToResource } from '~/store/navigation/navigation-action';

interface ResourceBreadcrumb extends Breadcrumb {
    uuid: string;
}

type BreadcrumbsDataProps = Pick<BreadcrumbsProps, 'items'>;
type BreadcrumbsActionProps = Pick<BreadcrumbsProps, 'onClick' | 'onContextMenu'>;

const memoizedMapStateToProps = () => {
    let items: ResourceBreadcrumb[] = [];
    return ({ router, treePicker }: RootState): BreadcrumbsDataProps => {
        if (router.location) {
            const projectMatch = matchProjectRoute(location.pathname);
            const collectionMatch = matchProjectRoute(location.pathname);
            const uuid = projectMatch && projectMatch.params.id
                || collectionMatch && collectionMatch.params.id
                || '';
            const tree = getTreePicker(SIDE_PANEL_TREE)(treePicker);
            if (tree) {
                const ancestors = getNodeAncestors(uuid)(tree);
                const node = getNode(uuid)(tree);
                const nodes = node ? [...ancestors, node] : ancestors;
                items = nodes.map(({ value }) =>
                    typeof value.value === 'string'
                        ? { label: value.value, uuid: value.nodeId }
                        : { label: value.value.name, uuid: value.value.uuid });
            }
        }
        return { items };
    };
};

const mapDispatchToProps = (dispatch: Dispatch): BreadcrumbsActionProps => ({
    onClick: ({ uuid }: ResourceBreadcrumb) => {
        dispatch<any>(navigateToResource(uuid));
    },
    onContextMenu: () => { return; }
});

export const Breadcrumbs = connect(memoizedMapStateToProps(), mapDispatchToProps)(BreadcrumbsComponent);