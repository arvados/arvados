import {GraphNode}                                                  from '../../graph/graph-node';
import {Workflow}                                                   from '../../graph/workflow';
import {SVGUtils}                                                   from '../../utils/svg-utils';
import {GraphChange, SVGPlugin}                                     from '../plugin';
import {
    StepModel,
    WorkflowInputParameterModel,
    WorkflowOutputParameterModel
} from "cwlts/models";

export class SVGArrangePlugin implements SVGPlugin {
    private workflow: Workflow;
    private svgRoot: SVGSVGElement;
    private onBeforeChange: () => void;
    private onAfterChange: (updates: NodePositionUpdates) => void;
    private triggerAfterRender: () => void;

    registerWorkflow(workflow: Workflow): void {
        this.workflow = workflow;
        this.svgRoot  = workflow.svgRoot;
    }


    registerOnBeforeChange(fn: (change: GraphChange) => void): void {
        this.onBeforeChange = () => fn({type: "arrange"});
    }

    registerOnAfterChange(fn: (change: GraphChange) => void): void {
        this.onAfterChange = () => fn({type: "arrange"});
    }

    registerOnAfterRender(fn: (change: GraphChange) => void): void {
        this.triggerAfterRender = () => fn({type: "arrange"});
    }

    afterRender(): void {
        const model     = this.workflow.model;
        const arr = [] as Array<WorkflowInputParameterModel | WorkflowOutputParameterModel | StepModel>;
        const drawables = arr.concat(
            model.steps || [],
            model.inputs || [],
            model.outputs || []
        );

        for (const node of drawables) {
            if (node.isVisible) {
                const missingCoordinate = isNaN(parseInt(node.customProps["sbg:x"], 10));
                if (missingCoordinate) {
                    this.arrange();
                    return;
                }
            }
        }
    }

    arrange() {

        this.onBeforeChange();

        // We need to reset all transformations on the workflow for now.
        // @TODO Make arranging work without this
        this.workflow.resetTransform();

        // We need main graph and dangling nodes separately, they will be distributed differently
        const {mainGraph, danglingNodes} = this.makeNodeGraphs();

        // Create an array of columns, each containing a list of NodeIOs
        const columns = this.distributeNodesIntoColumns(mainGraph);

        // Get total area in which we will fit the graph, and per-column dimensions
        const {distributionArea, columnDimensions} = this.calculateColumnSizes(columns);

        // This will be the vertical middle around which the graph should be centered
        const verticalBaseline = distributionArea.height / 2;

        let xOffset    = 0;
        let maxYOffset = 0;

        // Here we will store positions for each node that is to be updated.
        // This should then be emitted as an afterChange event.
        const nodePositionUpdates = {} as NodePositionUpdates;

        columns.forEach((column, index) => {
            const colSize = columnDimensions[index];
            let yOffset   = verticalBaseline - (colSize.height / 2) - column[0].rect.height / 2;

            column.forEach(node => {
                yOffset += node.rect.height / 2;

                const matrix = SVGUtils.createMatrix().translate(xOffset, yOffset);

                yOffset += node.rect.height / 2;

                if (yOffset > maxYOffset) {
                    maxYOffset = yOffset;
                }

                node.el.setAttribute("transform", SVGUtils.matrixToTransformAttr(matrix));

                nodePositionUpdates[node.connectionID] = {
                    x: matrix.e,
                    y: matrix.f
                };

            });

            xOffset += colSize.width;
        });

        const danglingNodeKeys = Object.keys(danglingNodes).sort((a, b) => {

            const aIsInput  = a.startsWith("out/");
            const aIsOutput = a.startsWith("in/");
            const bIsInput  = b.startsWith("out/");
            const bIsOutput = b.startsWith("in/");

            const lowerA = a.toLowerCase();
            const lowerB = b.toLowerCase();

            if (aIsOutput) {

                if (bIsOutput) {
                    return lowerB.localeCompare(lowerA);
                }
                else {
                    return 1;
                }
            } else if (aIsInput) {
                if (bIsOutput) {
                    return -1;
                }
                if (bIsInput) {
                    return lowerB.localeCompare(lowerA);
                }
                else {
                    return 1;
                }
            } else {
                if (!bIsOutput && !bIsInput) {
                    return lowerB.localeCompare(lowerA);
                }
                else {
                    return -1;
                }
            }
        });

        const danglingNodeMarginOffset = 30;
        const danglingNodeSideLength   = GraphNode.radius * 5;

        let maxNodeHeightInRow = 0;
        let row                = 0;
        const indexWidthMap      = new Map<number, number>();
        const rowMaxHeightMap    = new Map<number, number>();

        xOffset = 0;

        const danglingRowAreaWidth = Math.max(distributionArea.width, danglingNodeSideLength * 3);
        danglingNodeKeys.forEach((connectionID, index) => {
            const el   = danglingNodes[connectionID] as SVGGElement;
            const rect = el.firstElementChild!.getBoundingClientRect();
            indexWidthMap.set(index, rect.width);

            if (xOffset === 0) {
                xOffset -= rect.width / 2;
            }
            if (rect.height > maxNodeHeightInRow) {
                maxNodeHeightInRow = rect.height;
            }
            xOffset += rect.width + danglingNodeMarginOffset + Math.max(150 - rect.width, 0);

            if (xOffset >= danglingRowAreaWidth && index < danglingNodeKeys.length - 1) {
                rowMaxHeightMap.set(row++, maxNodeHeightInRow);
                maxNodeHeightInRow = 0;
                xOffset            = 0;
            }
        });

        rowMaxHeightMap.set(row, maxNodeHeightInRow);
        let colYOffset = maxYOffset;
        xOffset        = 0;
        row            = 0;

        danglingNodeKeys.forEach((connectionID, index) => {
            const el        = danglingNodes[connectionID] as SVGGElement;
            const width     = indexWidthMap.get(index)!;
            const rowHeight = rowMaxHeightMap.get(row)!;
            let left        = xOffset + width / 2;
            const top       = colYOffset
                + danglingNodeMarginOffset
                + Math.ceil(rowHeight / 2)
                + ((xOffset === 0 ? 0 : left) / danglingRowAreaWidth) * danglingNodeSideLength;

            if (xOffset === 0) {
                left -= width / 2;
                xOffset -= width / 2;
            }
            xOffset += width + danglingNodeMarginOffset + Math.max(150 - width, 0);

            const matrix = SVGUtils.createMatrix().translate(left, top);
            el.setAttribute("transform", SVGUtils.matrixToTransformAttr(matrix));

            nodePositionUpdates[connectionID] = {x: matrix.e, y: matrix.f};

            if (xOffset >= danglingRowAreaWidth) {
                colYOffset += Math.ceil(rowHeight) + danglingNodeMarginOffset;
                xOffset            = 0;
                maxNodeHeightInRow = 0;
                row++;
            }
        });

        this.workflow.redrawEdges();
        this.workflow.fitToViewport();

        this.onAfterChange(nodePositionUpdates);
        this.triggerAfterRender();

        for (const id in nodePositionUpdates) {
            const pos       = nodePositionUpdates[id];
            const nodeModel = this.workflow.model.findById(id);
            if (!nodeModel.customProps) {
                nodeModel.customProps = {};
            }

            Object.assign(nodeModel.customProps, {
                "sbg:x": pos.x,
                "sbg:y": pos.y
            });
        }

        return nodePositionUpdates;
    }

    /**
     * Calculates column dimensions and total graph area
     * @param {NodeIO[][]} columns
     */
    private calculateColumnSizes(columns: NodeIO[][]): {
        columnDimensions: {
            width: number,
            height: number
        }[],
        distributionArea: {
            width: number,
            height: number
        }
    } {
        const distributionArea = {width: 0, height: 0};
        const columnDimensions: any[] = [];

        for (let i = 1; i < columns.length; i++) {

            let width  = 0;
            let height = 0;

            for (let j = 0; j < columns[i].length; j++) {
                const entry = columns[i][j];

                height += entry.rect.height;

                if (width < entry.rect.width) {
                    width = entry.rect.width;
                }
            }

            columnDimensions[i] = {height, width};

            distributionArea.width += width;
            if (height > distributionArea.height) {
                distributionArea.height = height;
            }
        }

        return {
            columnDimensions,
            distributionArea
        };

    }

    /**
     * Maps node's connectionID to a 1-indexed column number
     */
    private distributeNodesIntoColumns(graph: NodeMap): Array<NodeIO[]> {
        const idToZoneMap   = {};
        const sortedNodeIDs = Object.keys(graph).sort((a, b) => b.localeCompare(a));
        const zones         = [] as any[];

        for (let i = 0; i < sortedNodeIDs.length; i++) {
            const nodeID = sortedNodeIDs[i];
            const node   = graph[nodeID];

            // For outputs and steps, we calculate the zone as a longest path you can take to them
            if (node.type !== "input") {
                idToZoneMap[nodeID] = this.traceLongestNodePathLength(node, graph);
            } else {
                //
                // Longest trace methods would put all inputs in the first column,
                // but we want it just behind the leftmost step that it is connected to
                // So instead of:
                //
                // (input)<----------------->(step)---
                // (input)<---------->(step)----------
                //
                // It should be:
                //
                // ---------------(input)<--->(step)---
                // --------(input)<-->(step)-----------
                //

                let closestNodeZone = Infinity;
                for (let i = 0; i < node.outputs.length; i++) {
                    const successorNodeZone = idToZoneMap[node.outputs[i]];

                    if (successorNodeZone < closestNodeZone) {
                        closestNodeZone = successorNodeZone;
                    }
                }
                if (closestNodeZone === Infinity) {
                    idToZoneMap[nodeID] = 1;
                } else {
                    idToZoneMap[nodeID] = closestNodeZone - 1;
                }

            }

            const zone = idToZoneMap[nodeID];
            zones[zone] || (zones[zone] = []);

            zones[zone].push(graph[nodeID]);
        }

        return zones;

    }

    /**
     * Finds all nodes in the graph, and indexes them by their "data-connection-id" attribute
     */
    private indexNodesByID(): { [dataConnectionID: string]: SVGGElement } {
        const indexed = {};
        const nodes   = this.svgRoot.querySelectorAll(".node");

        for (let i = 0; i < nodes.length; i++) {
            indexed[nodes[i].getAttribute("data-connection-id")!] = nodes[i];
        }

        return indexed;
    }

    /**
     * Finds length of the longest possible path from the graph root to a node.
     * Lengths are 1-indexed. When a node has no predecessors, it will have length of 1.
     */
    private traceLongestNodePathLength(node: NodeIO, nodeGraph: any, visited = new Set<NodeIO>()): number {

        visited.add(node);

        if (node.inputs.length === 0) {
            return 1;
        }

        const inputPathLengths: any[] = [];

        for (let i = 0; i < node.inputs.length; i++) {
            const el = nodeGraph[node.inputs[i]];

            if (visited.has(el)) {
                continue;
            }

            inputPathLengths.push(this.traceLongestNodePathLength(el, nodeGraph, visited));
        }

        return Math.max(...inputPathLengths) + 1;
    }

    private makeNodeGraphs(): {
        mainGraph: NodeMap,
        danglingNodes: { [nodeID: string]: SVGGElement }
    } {

        // We need all nodes in order to find the dangling ones, those will be sorted separately
        const allNodes = this.indexNodesByID();

        // Make a graph representation where you can trace inputs and outputs from/to connection ids
        const nodeGraph = {} as NodeMap;

        // Edges are the main source of information from which we will distribute nodes
        const edges = this.svgRoot.querySelectorAll(".edge");

        for (let i = 0; i < edges.length; i++) {

            const edge = edges[i];

            const sourceConnectionID      = edge.getAttribute("data-source-connection")!;
            const destinationConnectionID = edge.getAttribute("data-destination-connection")!;

            const [sourceSide, sourceNodeID, sourcePortID]                = sourceConnectionID.split("/");
            const [destinationSide, destinationNodeID, destinationPortID] = destinationConnectionID.split("/");

            // Both source and destination are considered to be steps by default
            let sourceType      = "step";
            let destinationType = "step";

            // Ports have the same node and port ids
            if (sourceNodeID === sourcePortID) {
                sourceType = sourceSide === "in" ? "output" : "input";
            }

            if (destinationNodeID === destinationPortID) {
                destinationType = destinationSide === "in" ? "output" : "input";
            }

            // Initialize keys on graph if they don't exist
            const sourceNode      = this.svgRoot.querySelector(`.node[data-id="${sourceNodeID}"]`) as SVGGElement;
            const destinationNode = this.svgRoot.querySelector(`.node[data-id="${destinationNodeID}"]`) as SVGGElement;

            const sourceNodeConnectionID      = sourceNode.getAttribute("data-connection-id")!;
            const destinationNodeConnectionID = destinationNode.getAttribute("data-connection-id")!;

            // Source and destination of this edge are obviously not dangling, so we can remove them
            // from the set of potentially dangling nodes
            delete allNodes[sourceNodeConnectionID];
            delete allNodes[destinationNodeConnectionID];

            // Ensure that the source node has its entry in the node graph
            (nodeGraph[sourceNodeID] || (nodeGraph[sourceNodeID] = {
                inputs: [],
                outputs: [],
                type: sourceType,
                connectionID: sourceNodeConnectionID,
                el: sourceNode,
                rect: sourceNode.getBoundingClientRect()
            }));

            // Ensure that the source node has its entry in the node graph
            (nodeGraph[destinationNodeID] || (nodeGraph[destinationNodeID] = {
                inputs: [],
                outputs: [],
                type: destinationType,
                connectionID: destinationNodeConnectionID,
                el: destinationNode,
                rect: destinationNode.getBoundingClientRect()
            }));

            nodeGraph[sourceNodeID].outputs.push(destinationNodeID);
            nodeGraph[destinationNodeID].inputs.push(sourceNodeID);
        }

        return {
            mainGraph: nodeGraph,
            danglingNodes: allNodes
        };
    }
}


export type NodeIO = {
    inputs: string[],
    outputs: string[],
    connectionID: string,
    el: SVGGElement,
    rect: ClientRect,
    type: "step" | "input" | "output" | string
};
export type NodeMap = { [connectionID: string]: NodeIO }

export type NodePositionUpdates = { [connectionID: string]: { x: number, y: number } };
