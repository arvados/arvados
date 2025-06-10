// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import Skeleton, { SkeletonProps } from 'react-loading-skeleton';

export const LoadingIndicator = (props: SkeletonProps) => (<Skeleton duration={1} {...props} />);
