// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { LinkResource } from 'models/link';
import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ServiceRepository, getResourceService } from 'services/services';
import { Resource, TrashableResource, extractUuidKind } from 'models/resource';
import { CommonResourceServiceError, getCommonResourceServiceError } from 'services/common-service/common-resource-service';

type NameableResource = Resource & { name?: string };

/**
 * Validates links are not to trashed resources and updates link resource names
 * to match resource name if necessary
 */
const verifyAndUpdateLink = async (link: LinkResource, dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<LinkResource | undefined> => {
    //head resource should already be in the store
    let headResource: Resource | undefined = getState().resources[link.headUuid];
    //If resource not in store, fetch it
    if (!headResource) {
        try {
            headResource = await fetchResource(link.headUuid)(dispatch, getState, services);
        } catch (e) {
            // If not found, assume deleted permanently and suppress this entry
            if (getCommonResourceServiceError(e) === CommonResourceServiceError.NOT_FOUND) {
                return undefined;
            }
            // If non-404 exception was raised, fall through to the headResource check
        }
        // Any other error we keep the entry but skip updating the name
        if (!headResource) {
            if (!link.name) console.error('Could not validate link', link, 'because link head', link.headUuid, 'is not available');
            return link;
        }
    }
    // If resource is trashed, filter it out
    if ((headResource as TrashableResource).isTrashed) {
        return undefined;
    }

    if (validateLinkNameProp(link, headResource) === true) return link;

    const updatedLink = updateLinkNameProp(link, headResource);
    updateRemoteLinkName(updatedLink)(dispatch, getState, services);

    return updatedLink;
};

/**
 * Filters links to trashed / 404ed resources and updates link name to match resource
 */
export const verifyAndUpdateLinks = async (links: LinkResource[], dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<LinkResource[]> => {
    // Verify and update links in paralell
    const updatedLinks = links.map((link) => verifyAndUpdateLink(link, dispatch, getState, services));
    // Filter out undefined links (trashed, malformed or 404)
    const validLinks = (await Promise.all(updatedLinks)).filter((link): link is LinkResource => (link !== undefined && !!link.headUuid));

    return Promise.resolve(validLinks);
};

/**
 * Fetches any resource type for verifying link names / trash status
 * Exposes exceptions to allow the consumer to differentiate errors
 */
const fetchResource = (uuid: string, showErrors?: boolean) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository): Promise<Resource | undefined> => {
    const kind = extractUuidKind(uuid);
    const service = getResourceService(kind)(services);
    if (service) {
        return await service.get(uuid, showErrors);
    }
    return undefined;
};

const validateLinkNameProp = (link: LinkResource, head: NameableResource) => {
    if (!link.name || link.name !== head.name) return false;
    return true;
};

const updateLinkNameProp = (link: LinkResource, head: NameableResource): LinkResource => {
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
