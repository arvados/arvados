// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { LinkResource } from 'models/link';
import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { getResourceService } from 'services/services';
import { Resource, extractUuidKind } from 'models/resource';

type NameableResource = Resource & { name?: string };

export const verifyAndUpdateLinkName = async (link: LinkResource, dispatch: Dispatch, getState: () => RootState, services: ServiceRepository):Promise<string> => {
  //check if head resource is already in the store
    let headResource: Resource | undefined = getState().resources[link.headUuid];
    if (!headResource) {
        headResource = await fetchResource(link.headUuid)(dispatch, getState, services);
        if(!headResource) {
            console.error('Could not verify link', link);
            return link.name;
        }
    }

    if (validateLinkNameProp(link, headResource) === true) return link.name;

    const updatedLink = updateLinkNameProp(link, headResource);

    return updatedLink.name;
};

const fetchResource = (uuid: string, showErrors?: boolean) => async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
    try {
        const kind = extractUuidKind(uuid);
        const service = getResourceService(kind)(services);
        if (service) {
            const resource = await service.get(uuid, showErrors);
            return resource;
        }
    } catch(e) {
        console.error(e);
    }
    return undefined;
};

const validateLinkNameProp = (link: LinkResource, head: NameableResource) => {
  if(!link.name || link.name !== head.name) return false;
    return true;
};

const updateLinkNameProp = (link: LinkResource, head: NameableResource) => {
  const updatedLink = {...link};
  if(head.name) updatedLink.name = head.name;
  return updatedLink;
}
