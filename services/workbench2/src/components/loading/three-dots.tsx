// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// The MIT License (MIT)

/*
 * This file includes code from react-loader-spinner, which is licensed under the MIT License.
 * Copyright (c) 2018 Mohan Pd.
 * https://github.com/mhnpd/react-loader-spinner
 * See the LICENSE file for more details.
 */


import React, { FunctionComponent } from 'react';
import { styled } from 'styled-components';

const DEFAULT_COLOR = '#4fa94d';

const DEFAULT_WAI_ARIA_ATTRIBUTE = {
    'aria-busy': true,
    role: 'progressbar',
};

type Style = {
    [key: string]: string;
};

interface PrimaryProps {
    height?: string | number;
    width?: string | number;
    ariaLabel?: string;
    wrapperStyle?: Style;
    wrapperClass?: string;
    visible?: boolean;
}

interface BaseProps extends PrimaryProps {
    color?: string;
}

//  changed from div to span to fix DOM nesting error
const SvgWrapper = styled.span<{ $visible: boolean }>`
    display: ${(props) => (props.$visible ? 'flex' : 'none')};
`;

interface ThreeDotsProps extends BaseProps {
    radius?: string | number;
}

export const ThreeDots: FunctionComponent<ThreeDotsProps> = ({
    height = 80,
    width = 80,
    radius = 9,
    color = DEFAULT_COLOR,
    ariaLabel = 'three-dots-loading',
    wrapperStyle,
    wrapperClass,
    visible = true,
}) => (
    <SvgWrapper
        style={wrapperStyle}
        $visible={visible}
        className={wrapperClass}
        data-testid='three-dots-loading'
        aria-label={ariaLabel}
        {...DEFAULT_WAI_ARIA_ATTRIBUTE}
    >
        <svg
            width={width}
            height={height}
            viewBox='0 0 120 30'
            xmlns={'http://www.w3.org/2000/svg'}
            fill={color}
            data-testid='three-dots-svg'
        >
            <circle
                cx='15'
                cy='15'
                r={Number(radius) + 6}
            >
                <animate
                    attributeName='r'
                    from='15'
                    to='15'
                    begin='0s'
                    dur='0.8s'
                    values='15;9;15'
                    calcMode='linear'
                    repeatCount='indefinite'
                />
                <animate
                    attributeName='fill-opacity'
                    from='1'
                    to='1'
                    begin='0s'
                    dur='0.8s'
                    values='1;.5;1'
                    calcMode='linear'
                    repeatCount='indefinite'
                />
            </circle>
            <circle
                cx='60'
                cy='15'
                r={radius}
                attributeName='fill-opacity'
                from='1'
                to='0.3'
            >
                <animate
                    attributeName='r'
                    from='9'
                    to='9'
                    begin='0s'
                    dur='0.8s'
                    values='9;15;9'
                    calcMode='linear'
                    repeatCount='indefinite'
                />
                <animate
                    attributeName='fill-opacity'
                    from='0.5'
                    to='0.5'
                    begin='0s'
                    dur='0.8s'
                    values='.5;1;.5'
                    calcMode='linear'
                    repeatCount='indefinite'
                />
            </circle>
            <circle
                cx='105'
                cy='15'
                r={Number(radius) + 6}
            >
                <animate
                    attributeName='r'
                    from='15'
                    to='15'
                    begin='0s'
                    dur='0.8s'
                    values='15;9;15'
                    calcMode='linear'
                    repeatCount='indefinite'
                />
                <animate
                    attributeName='fill-opacity'
                    from='1'
                    to='1'
                    begin='0s'
                    dur='0.8s'
                    values='1;.5;1'
                    calcMode='linear'
                    repeatCount='indefinite'
                />
            </circle>
        </svg>
    </SvgWrapper>
);
