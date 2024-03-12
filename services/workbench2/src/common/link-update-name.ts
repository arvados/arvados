// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { LinkResource } from 'models/link';
import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ServiceRepository, getResourceService } from 'services/services';
import { Resource, extractUuidKind } from 'models/resource';

type NameableResource = Resource & { name?: string };

export const verifyAndUpdateLink = async (link: LinkResource, dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<LinkResource> => {
    //check if head resource is already in the store
    let headResource: Resource | undefined = getState().resources[link.headUuid];
    //if not, fetch it
    if (!headResource) {
        headResource = await fetchResource(link.headUuid)(dispatch, getState, services);
        if (!headResource) {
            if (!link.name) console.error('Could not validate link', link, 'because link head', link.headUuid, 'is not available');
            return link;
        }
    }

    if (validateLinkNameProp(link, headResource) === true) return link;

    const updatedLink = updateLinkNameProp(link, headResource);
    updateRemoteLinkName(updatedLink)(dispatch, getState, services);

    return updatedLink;
};

export const verifyAndUpdateLinks = async (links: LinkResource[], dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    const updatedLinks = links.map((link) => verifyAndUpdateLink(link, dispatch, getState, services));
        return Promise.all(updatedLinks);
};

const fetchResource = (uuid: string, showErrors?: boolean) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    try {
        const kind = extractUuidKind(uuid);
        const service = getResourceService(kind)(services);
        if (service) {
            const resource = await service.get(uuid, showErrors);
            return resource;
        }
    } catch (e) {
        console.error(`Could not fetch resource ${uuid}`, e);
    }
    return undefined;
};

const validateLinkNameProp = (link: LinkResource, head: NameableResource) => {
    if (!link.name || link.name !== head.name) return false;
    return true;
};

const updateLinkNameProp = (link: LinkResource, head: NameableResource) => {
    const updatedLink = { ...link };
    if (head.name) updatedLink.name = head.name;
    return updatedLink;
};

const updateRemoteLinkName = (link: LinkResource) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    try {
        const kind = extractUuidKind(link.uuid);
        const service = getResourceService(kind)(services);
        if (service) {
            service.update(link.uuid, { name: link.name });
        }
    } catch (error) {
        console.error('Could not update link name', link, error);
    }
};
