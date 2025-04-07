// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Card } from '@mui/material';
import { DetailsCardRoot } from "views-components/details-card/details-card-root";
import { CollectionPanelFiles } from 'views-components/collection-panel-files/collection-panel-files';

export const CollectionPanel = () => {
    return <div style={{width: "100%", height: "100%"}}>
        <DetailsCardRoot />
        <Card >
            <CollectionPanelFiles isWritable={false} />
        </Card>
    </div>;
};