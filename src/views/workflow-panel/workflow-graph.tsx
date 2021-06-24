// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WorkflowResource } from "models/workflow";
import { WorkflowFactory } from "cwlts/models";
import * as yaml from 'js-yaml';
import "lib/cwl-svg/assets/styles/themes/rabix-dark/theme.css";
import "lib/cwl-svg/plugins/port-drag/theme.dark.css";
import "lib/cwl-svg/plugins/selection/theme.dark.css";
import {
    SelectionPlugin,
    SVGArrangePlugin,
    SVGEdgeHoverPlugin,
    SVGNodeMovePlugin,
    SVGPortDragPlugin, Workflow,
    ZoomPlugin
} from "lib/cwl-svg";

interface WorkflowGraphProps {
    workflow: WorkflowResource;
}
export class WorkflowGraph extends React.Component<WorkflowGraphProps, {}> {
    private svgRoot: React.RefObject<SVGSVGElement> = React.createRef();

    setGraph() {
        const graphs = yaml.safeLoad(this.props.workflow.definition, { json: true });

        let workflowGraph = graphs;
        if (graphs.$graph) {
          workflowGraph = graphs.$graph.find((g: any) => g.class === 'Workflow');
        }

        const model = WorkflowFactory.from(workflowGraph);

        const workflow = new Workflow({
            model,
            svgRoot: this.svgRoot.current!,
            plugins: [
                new SVGArrangePlugin(),
                new SVGEdgeHoverPlugin(),
                new SVGNodeMovePlugin({
                    movementSpeed: 2
                }),
                new SVGPortDragPlugin(),
                new SelectionPlugin(),
                new ZoomPlugin(),
            ]
        });
        workflow.draw();
    }

    componentDidMount() {
        this.setGraph();
    }

    componentDidUpdate() {
        this.setGraph();
    }

    render() {
        return <svg
            ref={this.svgRoot}
            className="cwl-workflow"
            style={{
                width: '100%',
                height: '100%'
            }}
        />;
    }
}
