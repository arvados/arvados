// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';
import { bindDataExplorerActions } from '~/store/data-explorer/data-explorer-action';
import { setBreadcrumbs } from '~/store/breadcrumbs/breadcrumbs-actions';
import { dialogActions } from '~/store/dialog/dialog-actions';
import { LinkResource } from '~/models/link';
import { getResource } from '~/store/resources/resources';
import {snackbarActions, SnackbarKind} from '~/store/snackbar/snackbar-actions';

export const LINK_PANEL_ID = "linkPanelId";
export const linkPanelActions = bindDataExplorerActions(LINK_PANEL_ID);

export const LINK_REMOVE_DIALOG = 'linkRemoveDialog';
export const LINK_ATTRIBUTES_DIALOG = 'linkAttributesDialog';

export const loadLinkPanel = () =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(setBreadcrumbs([{ label: 'Links' }]));
        dispatch(linkPanelActions.REQUEST_ITEMS());
    };

export const openLinkAttributesDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const { resources } = getState();
        const link = getResource<LinkResource>(uuid)(resources);
        dispatch(dialogActions.OPEN_DIALOG({ id: LINK_ATTRIBUTES_DIALOG, data: { link } }));
    };

export const openLinkRemoveDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: LINK_REMOVE_DIALOG,
            data: {
                title: 'Remove link',
                text: 'Are you sure you want to remove this link?',
                confirmButtonLabel: 'Remove',
                uuid
            }
        }));
    };

export const removeLink = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...', kind: SnackbarKind.INFO }));
        try {
            await services.linkService.delete(uuid);
            dispatch(linkPanelActions.REQUEST_ITEMS());
            dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Link has been successfully removed.', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        } catch (e) {
            return;
        }
    };