// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Chip } from '@material-ui/core';
import { connect } from 'react-redux';
import { RootState } from 'store/store';
import CopyToClipboard from 'react-copy-to-clipboard';
import { getVocabulary } from 'store/vocabulary/vocabulary-selectors';
import { Dispatch } from 'redux';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { getTagValueLabel, getTagKeyLabel, Vocabulary } from 'models/vocabulary';

interface PropertyChipComponentDataProps {
    propKey: string;
    propValue: string;
    className: string;
    vocabulary: Vocabulary;
}

interface PropertyChipComponentActionProps {
    onDelete?: () => void;
    onCopy: (message: string) => void;
}

type PropertyChipComponentProps = PropertyChipComponentActionProps & PropertyChipComponentDataProps;

const mapStateToProps = ({ properties }: RootState) => ({
    vocabulary: getVocabulary(properties),
});

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onCopy: (message: string) => dispatch(snackbarActions.OPEN_SNACKBAR({
        message,
        hideDuration: 2000,
        kind: SnackbarKind.SUCCESS
    }))
});

// Renders a Chip with copyable-on-click tag:value data based on the vocabulary
export const PropertyChipComponent = connect(mapStateToProps, mapDispatchToProps)(
    ({ propKey, propValue, vocabulary, className, onCopy, onDelete }: PropertyChipComponentProps) => {
        const label = `${getTagKeyLabel(propKey, vocabulary)}: ${getTagValueLabel(propKey, propValue, vocabulary)}`;
        return (
            <CopyToClipboard key={propKey} text={label} onCopy={() => onCopy("Copied to clipboard")}>
                <Chip onDelete={onDelete} key={propKey}
                    className={className} label={label} />
            </CopyToClipboard>
        );
    }
);

export const getPropertyChip = (k: string, v: string, handleDelete: any, className: string) =>
    <PropertyChipComponent
        key={`${k}-${v}`} className={className}
        onDelete={handleDelete}
        propKey={k} propValue={v} />;
