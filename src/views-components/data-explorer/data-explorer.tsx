// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { RootState } from "../../store/store";
import DataExplorer from "../../components/data-explorer/data-explorer";
import { getDataExplorer } from "../../store/data-explorer/data-explorer-reducer";

export default connect((state: RootState, props: { id: string }) =>
    getDataExplorer(state.dataExplorer, props.id)
)(DataExplorer);
