import {ParameterTypeModel, StepModel, WorkflowInputParameterModel, WorkflowOutputParameterModel} from "cwlts/models";
import {HtmlUtils} from "../utils/html-utils";
import {SVGUtils} from "../utils/svg-utils";
import {IOPort} from "./io-port";

export type NodePosition = { x: number, y: number };
export type NodeDataModel = WorkflowInputParameterModel | WorkflowOutputParameterModel | StepModel;

export class GraphNode {

    public position: NodePosition = {x: 0, y: 0};

    static radius = 30;

    constructor(position: Partial<NodePosition>,
                private dataModel: NodeDataModel) {

        this.dataModel = dataModel;

        Object.assign(this.position, position);
    }

    /**
     * @FIXME Making icons increases the rendering time by 50-100%. Try embedding the SVG directly.
     */

    private static workflowIconSvg: string   = "<svg class=\"node-icon\" xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 400.01 399.88\" x=\"-9\" y=\"-10\" width=\"20\" height=\"20\"><title>workflow</title><path d=\"M400,200a80,80,0,0,1-140.33,52.53L158.23,303.24a80,80,0,1,1-17.9-35.77l101.44-50.71a80.23,80.23,0,0,1,0-33.52L140.33,132.53a79.87,79.87,0,1,1,17.9-35.77l101.44,50.71A80,80,0,0,1,400,200Z\" transform=\"translate(0.01 -0.16)\"/></svg>";
    private static toolIconSvg: string       = "<svg class=\"node-icon\" xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 398.39 397.78\" x=\"-10\" y=\"-8\" width=\"20\" height=\"15\"><title>tool2</title><polygon points=\"38.77 397.57 0 366 136.15 198.78 0 31.57 38.77 0 200.63 198.78 38.77 397.57\"/><rect x=\"198.39\" y=\"347.78\" width=\"200\" height=\"50\"/></svg>";
    private static fileInputIconSvg: string  = "<svg class=\"node-icon\" xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 499 462.86\" y=\"-10\" x=\"-11\" width=\"20\" height=\"20\"><title>file_input</title><path d=\"M386.06,0H175V58.29l50,50V50H337.81V163.38h25l86.19.24V412.86H225V353.71l-50,50v59.15H499V112.94Zm1.75,113.45v-41l41.1,41.1Z\"/><polygon points=\"387.81 1.06 387.81 1.75 387.12 1.06 387.81 1.06\"/><polygon points=\"290.36 231 176.68 344.68 141.32 309.32 194.64 256 0 256 0 206 194.64 206 142.32 153.68 177.68 118.32 290.36 231\"/></svg>";
    private static fileOutputIconSvg: string = "<svg class=\"node-icon\" xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 499 462.86\" x=\"-7\" y=\"-11\" width=\"20\" height=\"20\"><title>file_output</title><polygon points=\"387.81 1.06 387.81 1.75 387.12 1.06 387.81 1.06\"/><polygon points=\"499 231 385.32 344.68 349.96 309.32 403.28 256 208.64 256 208.64 206 403.28 206 350.96 153.68 386.32 118.32 499 231\"/><path d=\"M187.81,163.38l77.69.22H324V112.94L211.06,0H0V462.86H324V298.5H274V412.86H50V50H162.81V163.38Zm25-90.92,41.1,41.1-41.1-.11Z\"/></svg>";
    private static inputIconSvg: string      = "<svg class=\"node-icon\" xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 499 365\" x=\"-11\" y=\"-10\" width=\"20\" height=\"20\"><title>type_input</title><g id=\"input\"><path d=\"M316.5,68a181.72,181.72,0,0,0-114.12,40.09L238,143.72a132.5,132.5,0,1,1,1.16,214.39L203.48,393.8A182.5,182.5,0,1,0,316.5,68Z\" transform=\"translate(0 -68)\"/><g id=\"Layer_22\" data-name=\"Layer 22\"><g id=\"Layer_9_copy_4\" data-name=\"Layer 9 copy 4\"><polygon points=\"290.36 182 176.68 295.68 141.32 260.32 194.64 207 0 207 0 157 194.64 157 142.32 104.68 177.68 69.32 290.36 182\"/></g></g></g></svg>";
    private static outputIconSvg: string     = "<svg class=\"node-icon\" xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 500.36 365\" x=\"-9\" y=\"-10\" width=\"20\" height=\"20\"><title>type_output</title><g id=\"output\"><path d=\"M291.95,325.23a134,134,0,0,1-15.76,19,132.5,132.5,0,1,1,0-187.38,133.9,133.9,0,0,1,16.16,19.55l35.81-35.81A182.5,182.5,0,1,0,327.73,361Z\" transform=\"translate(0 -68)\"/><g id=\"circle_source_copy\" data-name=\"circle source copy\"><g id=\"Layer_22_copy\" data-name=\"Layer 22 copy\"><g id=\"Layer_9_copy_5\" data-name=\"Layer 9 copy 5\"><polygon points=\"500.36 182 386.68 295.68 351.32 260.32 404.64 207 210 207 210 157 404.64 157 352.32 104.68 387.68 69.32 500.36 182\"/></g></g></g></g></svg>";

    private static makeIconFragment(model: any) {

        let iconStr = "";

        if (model instanceof StepModel && model.run) {

            if (model.run.class === "Workflow") {
                iconStr = this.workflowIconSvg;
            } else if (model.run.class === "CommandLineTool") {
                iconStr = this.toolIconSvg;
            }

        } else if (model instanceof WorkflowInputParameterModel && model.type) {
            if (model.type.type === "File" || (model.type.type === "array" && model.type.items === "File")) {
                iconStr = this.fileInputIconSvg;
            } else {
                iconStr = this.inputIconSvg;
            }
        } else if (model instanceof WorkflowOutputParameterModel && model.type) {
            if (model.type.type === "File" || (model.type.type === "array" && model.type.items === "File")) {
                iconStr = this.fileOutputIconSvg;
            } else {
                iconStr = this.outputIconSvg;
            }
        }

        return iconStr;
    }

    static makeTemplate(dataModel: {
        id: string,
        connectionId: string,
        label?: string,
        in?: any[],
        type?: ParameterTypeModel
        out?: any[],
        customProps?: {
            "sbg:x"?: number
            "sbg:y"?: number
        }
    }, labelScale = 1): string {

        const x = ~~(dataModel.customProps && dataModel.customProps["sbg:x"])!;
        const y = ~~(dataModel.customProps && dataModel.customProps["sbg:y"])!;

        let nodeTypeClass = "step";
        if (dataModel instanceof WorkflowInputParameterModel) {
            nodeTypeClass = "input";
        } else if (dataModel instanceof WorkflowOutputParameterModel) {
            nodeTypeClass = "output";
        }

        const inputs   = (dataModel.in || []).filter(p => p.isVisible);
        const outputs  = (dataModel.out || []).filter(p => p.isVisible);
        const maxPorts = Math.max(inputs.length, outputs.length);
        const radius   = GraphNode.radius + maxPorts * IOPort.radius;

        let typeClass = "";
        let itemsClass = "";

        if (dataModel.type) {
            typeClass = "type-" + dataModel.type.type;

            if(dataModel.type.items){
                itemsClass = "items-" + dataModel.type.items;
            }
        }

        const inputPortTemplates = inputs
            .sort((a, b) => -a.id.localeCompare(b.id))
            .map((p, i, arr) => GraphNode.makePortTemplate(
                p,
                "input",
                SVGUtils.matrixToTransformAttr(
                    GraphNode.createPortMatrix(arr.length, i, radius, "input")
                )
            ))
            .reduce((acc, tpl) => acc + tpl, "");

        const outputPortTemplates = outputs
            .sort((a, b) => -a.id.localeCompare(b.id))
            .map((p, i, arr) => GraphNode.makePortTemplate(
                p,
                "output",
                SVGUtils.matrixToTransformAttr(
                    GraphNode.createPortMatrix(arr.length, i, radius, "output")
                )
            ))
            .reduce((acc, tpl) => acc + tpl, "");

        return `
            <g tabindex="-1" class="node ${nodeTypeClass} ${typeClass} ${itemsClass}"
               data-connection-id="${dataModel.connectionId}"
               transform="matrix(1, 0, 0, 1, ${x}, ${y})"
               data-id="${dataModel.id}">
               
                <g class="core" transform="matrix(1, 0, 0, 1, 0, 0)">
                    <circle cx="0" cy="0" r="${radius}" class="outer"></circle>
                    <circle cx="0" cy="0" r="${radius * .75}" class="inner"></circle>
                    
                    ${GraphNode.makeIconFragment(dataModel)}
                </g>
                
                <text transform="matrix(${labelScale},0,0,${labelScale},0,${radius + 30})" class="title label">${HtmlUtils.escapeHTML(dataModel.label || dataModel.id)}</text>
                
                ${inputPortTemplates}
                ${outputPortTemplates}
            </g>
        `;
    }

    private static makePortTemplate(port: {
                                        label?: string,
                                        id: string,
                                        connectionId: string
                                    },
                                    type: "input" | "output",
                                    transform = "matrix(1, 0, 0, 1, 0, 0)"): string {

        const portClass = type === "input" ? "input-port" : "output-port";
        const label     = port.label || port.id;

        return `
            <g class="port ${portClass}" transform="${transform || "matrix(1, 0, 0, 1, 0, 0)"}"
               data-connection-id="${port.connectionId}"
               data-port-id="${port.id}"
            >
                <g class="io-port">
                    <circle cx="0" cy="0" r="7" class="port-handle"></circle>
                </g>
                <text x="0" y="0" transform="matrix(1,0,0,1,0,0)" class="label unselectable">${label}</text>
            </g>
            
        `;
    }

    public static createPortMatrix(totalPortLength: number,
                                   portIndex: number,
                                   radius: number,
                                   type: "input" | "output"): SVGMatrix {
        const availableAngle = 140;

        let rotationAngle =
                // Starting rotation angle
                (-availableAngle / 2) +
                (
                    // Angular offset by element index
                    (portIndex + 1)
                    // Angle between elements
                    * availableAngle / (totalPortLength + 1)
                );

        if (type === "input") {
            rotationAngle =
                // Determines the starting rotation angle
                180 - (availableAngle / -2)
                // Determines the angular offset modifier for the current index
                - (portIndex + 1)
                // Determines the angular offset
                * availableAngle / (totalPortLength + 1);
        }

        const matrix = SVGUtils.createMatrix();
        return matrix.rotate(rotationAngle).translate(radius, 0).rotate(-rotationAngle);
    }

    static patchModelPorts<T>(model: T & { connectionId: string, id: string }): T {
        const patch = [{connectionId: model.connectionId, isVisible: true, id: model.id}];
        if (model instanceof WorkflowInputParameterModel) {
            const copy = Object.create(model);
            return Object.assign(copy, {out: patch});


        } else if (model instanceof WorkflowOutputParameterModel) {
            const copy = Object.create(model);
            return Object.assign(copy, {in: patch});
        }

        return model;
    }

}
