import {WorkflowStepInputModel}       from "cwlts/models/generic";
import {StepModel}                    from "cwlts/models/generic/StepModel";
import {WorkflowInputParameterModel}  from "cwlts/models/generic/WorkflowInputParameterModel";
import {WorkflowModel}                from "cwlts/models/generic/WorkflowModel";
import {WorkflowOutputParameterModel} from "cwlts/models/generic/WorkflowOutputParameterModel";
import {SVGPlugin}                    from "../plugins/plugin";
import {DomEvents}                    from "../utils/dom-events";
import {EventHub}                     from "../utils/event-hub";
import {Connectable}                  from "./connectable";
import {Edge as GraphEdge}            from "./edge";
import {GraphNode}                    from "./graph-node";
import {StepNode}                     from "./step-node";
import {TemplateParser}               from "./template-parser";
import {WorkflowStepOutputModel}      from "cwlts/models";

/**
 * @FIXME validation states of old and newly created edges
 */
export class Workflow {

    readonly eventHub: EventHub;
    readonly svgID = this.makeID();

    minScale = 0.2;
    maxScale = 2;

    domEvents: DomEvents;
    svgRoot: SVGSVGElement;
    workflow: SVGGElement;
    model: WorkflowModel;
    editingEnabled = true;

    /** Scale of labels, they are different than scale of other elements in the workflow */
    labelScale = 1;

    private workflowBoundingClientRect: any;
    private plugins: SVGPlugin[]  = [];
    private disposers: Function[] = [];

    private pendingFirstDraw = true;

    /** Stored in order to ensure that once destroyed graph cannot be reused again */
    private isDestroyed = false;

    constructor(parameters: {
        svgRoot: SVGSVGElement,
        model: WorkflowModel,
        plugins?: SVGPlugin[],
        editingEnabled?: boolean
    }) {
        this.svgRoot        = parameters.svgRoot;
        this.plugins        = parameters.plugins || [];
        this.domEvents      = new DomEvents(this.svgRoot as any);
        this.model          = parameters.model;
        this.editingEnabled = parameters.editingEnabled !== false; // default to true if undefined

        this.svgRoot.classList.add(this.svgID);

        this.svgRoot.innerHTML = `
            <rect x="0" y="0" width="100%" height="100%" class="pan-handle" transform="matrix(1,0,0,1,0,0)"></rect>
            <g class="workflow" transform="matrix(1,0,0,1,0,0)"></g>
        `;

        this.workflow = this.svgRoot.querySelector(".workflow") as any;

        this.invokePlugins("registerWorkflow", this);

        this.eventHub = new EventHub([
            "connection.create",
            "app.create.step",
            "app.create.input",
            "app.create.output",
            "beforeChange",
            "afterChange",
            "afterRender",
            "selectionChange"
        ]);

        this.hookPlugins();
        this.draw(parameters.model);


        this.eventHub.on("afterRender", () => this.invokePlugins("afterRender"));
    }

    /** Current scale of the document */
    private docScale = 1;

    get scale() {
        return this.docScale;
    }

    // noinspection JSUnusedGlobalSymbols
    set scale(scale: number) {
        this.workflowBoundingClientRect = this.svgRoot.getBoundingClientRect();

        const x = (this.workflowBoundingClientRect.right + this.workflowBoundingClientRect.left) / 2;
        const y = (this.workflowBoundingClientRect.top + this.workflowBoundingClientRect.bottom) / 2;

        this.scaleAtPoint(scale, x, y);
    }

    static canDrawIn(element: SVGElement): boolean {
        return element.getBoundingClientRect().width !== 0;
    }

    static makeConnectionPath(x1: any, y1: any, x2: any, y2: any, forceDirection: "right" | "left" | string = "right"): string | undefined {

        if (!forceDirection) {
            return `M ${x1} ${y1} C ${(x1 + x2) / 2} ${y1} ${(x1 + x2) / 2} ${y2} ${x2} ${y2}`;
        } else if (forceDirection === "right") {
            const outDir = x1 + Math.abs(x1 - x2) / 2;
            const inDir  = x2 - Math.abs(x1 - x2) / 2;

            return `M ${x1} ${y1} C ${outDir} ${y1} ${inDir} ${y2} ${x2} ${y2}`;
        } else if (forceDirection === "left") {
            const outDir = x1 - Math.abs(x1 - x2) / 2;
            const inDir  = x2 + Math.abs(x1 - x2) / 2;

            return `M ${x1} ${y1} C ${outDir} ${y1} ${inDir} ${y2} ${x2} ${y2}`;
        }
        return undefined;
    }

    draw(model: WorkflowModel = this.model) {

        this.assertNotDestroyed("draw");

        // We will need to restore the transformations when we redraw the model, so save the current state
        const oldTransform = this.workflow.getAttribute("transform");

        const modelChanged = this.model !== model;

        if (modelChanged || this.pendingFirstDraw) {
            this.pendingFirstDraw = false;

            this.model = model;

            const stepChangeDisposer        = this.model.on("step.change", this.onStepChange.bind(this));
            const stepCreateDisposer        = this.model.on("step.create", this.onStepCreate.bind(this));
            const stepRemoveDisposer        = this.model.on("step.remove", this.onStepRemove.bind(this));
            const inputCreateDisposer       = this.model.on("input.create", this.onInputCreate.bind(this));
            const inputRemoveDisposer       = this.model.on("input.remove", this.onInputRemove.bind(this));
            const outputCreateDisposer      = this.model.on("output.create", this.onOutputCreate.bind(this));
            const outputRemoveDisposer      = this.model.on("output.remove", this.onOutputRemove.bind(this));
            const stepInPortShowDisposer    = this.model.on("step.inPort.show", this.onInputPortShow.bind(this));
            const stepInPortHideDisposer    = this.model.on("step.inPort.hide", this.onInputPortHide.bind(this));
            const connectionCreateDisposer  = this.model.on("connection.create", this.onConnectionCreate.bind(this));
            const connectionRemoveDisposer  = this.model.on("connection.remove", this.onConnectionRemove.bind(this));
            const stepOutPortCreateDisposer = this.model.on("step.outPort.create", this.onOutputPortCreate.bind(this));
            const stepOutPortRemoveDisposer = this.model.on("step.outPort.remove", this.onOutputPortRemove.bind(this));

            this.disposers.push(() => {
                stepChangeDisposer.dispose();
                stepCreateDisposer.dispose();
                stepRemoveDisposer.dispose();
                inputCreateDisposer.dispose();
                inputRemoveDisposer.dispose();
                outputCreateDisposer.dispose();
                outputRemoveDisposer.dispose();
                stepInPortShowDisposer.dispose();
                stepInPortHideDisposer.dispose();
                connectionCreateDisposer.dispose();
                connectionRemoveDisposer.dispose();
                stepOutPortCreateDisposer.dispose();
                stepOutPortRemoveDisposer.dispose();
            });

            this.invokePlugins("afterModelChange");
        }

        this.clearCanvas();

        const nodes = [
            ...this.model.steps,
            ...this.model.inputs,
            ...this.model.outputs
        ].filter(n => n.isVisible);

        /**
         * If there is a missing sbg:x or sbg:y property on any node model,
         * graph should be arranged to avoid random placement.
         */
        let nodeTemplate = "";

        for (const node of nodes) {
            const patched  = GraphNode.patchModelPorts(node);
            nodeTemplate += GraphNode.makeTemplate(patched);
        }

        this.workflow.innerHTML += nodeTemplate;

        this.redrawEdges();

        Array.from(this.workflow.querySelectorAll(".node")).forEach(e => {
            this.workflow.appendChild(e);
        });

        this.addEventListeners();

        this.workflow.setAttribute("transform", oldTransform!);

        this.scaleAtPoint(this.scale);


        this.invokePlugins("afterRender");
    }

    findParent(el: Element, parentClass = "node"): SVGGElement | undefined {
        let parentNode: Element | null = el;
        while (parentNode) {
            if (parentNode.classList.contains(parentClass)) {
                return parentNode as SVGGElement;
            }
            parentNode = parentNode.parentElement;
        }
        return undefined;
    }

    /**
     * Retrieves a plugin instance
     * @param {{new(...args: any[]) => T}} plugin
     * @returns {T}
     */
    getPlugin<T extends SVGPlugin>(plugin: { new(...args: any[]): T }): T {
        return this.plugins.find(p => p instanceof plugin) as T;
    }

    on(event: string, handler: any) {
        this.eventHub.on(event, handler);
    }

    off(event: string, handler: any) {
        this.eventHub.off(event, handler);
    }

    /**
     * Scales the workflow to fit the available viewport
     */
    fitToViewport(ignoreScaleLimits = false): void {

        this.scaleAtPoint(1);

        Object.assign(this.workflow.transform.baseVal.getItem(0).matrix, {
            e: 0,
            f: 0
        });

        const clientBounds = this.svgRoot.getBoundingClientRect();
        const wfBounds     = this.workflow.getBoundingClientRect();
        const padding    = 100;

        if (clientBounds.width === 0 || clientBounds.height === 0) {
            throw new Error("Cannot fit workflow to the area that has no visible viewport.");
        }

        const verticalScale   = (wfBounds.height) / (clientBounds.height - padding);
        const horizontalScale = (wfBounds.width) / (clientBounds.width - padding);

        const scaleFactor = Math.max(verticalScale, horizontalScale);

        // Cap the upscaling to 1, we don't want to zoom in workflows that would fit anyway
        let newScale = Math.min(this.scale / scaleFactor, 1);

        if (!ignoreScaleLimits) {
            newScale = Math.max(newScale, this.minScale);
        }

        this.scaleAtPoint(newScale);

        const scaledWFBounds = this.workflow.getBoundingClientRect();

        const moveY = clientBounds.top - scaledWFBounds.top + Math.abs(clientBounds.height - scaledWFBounds.height) / 2;
        const moveX = clientBounds.left - scaledWFBounds.left + Math.abs(clientBounds.width - scaledWFBounds.width) / 2;

        const matrix = this.workflow.transform.baseVal.getItem(0).matrix;
        matrix.e += moveX;
        matrix.f += moveY;
    }

    redrawEdges() {

        const highlightedEdges = new Set();

        Array.from(this.workflow.querySelectorAll(".edge")).forEach((el) => {
            if (el.classList.contains("highlighted")) {
                const edgeID = el.attributes["data-source-connection"].value + el.attributes["data-destination-connection"].value;
                highlightedEdges.add(edgeID);
            }
            el.remove();
        });


        const edgesTpl = this.model.connections
            .map(c => {
                const edgeId     = c.source.id + c.destination.id;
                const edgeStates = highlightedEdges.has(edgeId) ? "highlighted" : "";
                return GraphEdge.makeTemplate(c, this.workflow, edgeStates);
            })
            .reduce((acc, tpl) => acc! + tpl, "");

        this.workflow.innerHTML = edgesTpl + this.workflow.innerHTML;
    }

    /**
     * Scale the workflow by the scaleCoefficient (not compounded) over given coordinates
     */
    scaleAtPoint(scale = 1, x = 0, y = 0): void {

        this.docScale     = scale;
        this.labelScale = 1 + (1 - this.docScale) / (this.docScale * 2);

        const transform         = this.workflow.transform.baseVal;
        const matrix: SVGMatrix = transform.getItem(0).matrix;

        const coords = this.transformScreenCTMtoCanvas(x, y);

        matrix.e += matrix.a * coords.x;
        matrix.f += matrix.a * coords.y;
        matrix.a = matrix.d = scale;
        matrix.e -= scale * coords.x;
        matrix.f -= scale * coords.y;

        const nodeLabels: any = this.workflow.querySelectorAll(".node .label") as  NodeListOf<SVGPathElement>;

        for (const el of nodeLabels) {
            const matrix = el.transform.baseVal.getItem(0).matrix;

            Object.assign(matrix, {
                a: this.labelScale,
                d: this.labelScale
            });
        }

    }

    transformScreenCTMtoCanvas(x: any, y: any) {
        const svg   = this.svgRoot;
        const ctm   = this.workflow.getScreenCTM()!;
        const point = svg.createSVGPoint();
        point.x     = x;
        point.y     = y;

        const t = point.matrixTransform(ctm.inverse());
        return {
            x: t.x,
            y: t.y
        };
    }

    enableEditing(enabled: boolean): void {
        this.invokePlugins("onEditableStateChange", enabled);
        this.editingEnabled = enabled;
    }

    // noinspection JSUnusedGlobalSymbols
    destroy() {

        this.svgRoot.classList.remove(this.svgID);

        this.clearCanvas();
        this.eventHub.empty();

        this.invokePlugins("destroy");

        for (const dispose of this.disposers) {
            dispose();
        }

        this.isDestroyed = true;
    }

    resetTransform() {
        this.workflow.setAttribute("transform", "matrix(1,0,0,1,0,0)");
        this.scaleAtPoint();
    }

    private assertNotDestroyed(method: string) {
        if (this.isDestroyed) {
            throw new Error("Cannot call the " + method + " method on a destroyed graph. " +
                "Destroying this object removes DOM listeners, " +
                "and reusing it would result in unexpected things not working. " +
                "Instead, you can just call the “draw” method with a different model, " +
                "or create a new Workflow object.");

        }
    }

    private addEventListeners(): void {


        /**
         * Attach canvas panning
         */
        {
            let pane: SVGGElement | undefined;
            let x = 0;
            let y = 0;
            let matrix: SVGMatrix | undefined;
            this.domEvents.drag(".pan-handle", (dx, dy) => {

                matrix!.e = x + dx;
                matrix!.f = y + dy;

            }, (ev, el, root) => {
                pane   = root!.querySelector(".workflow") as SVGGElement;
                matrix = pane.transform.baseVal.getItem(0).matrix;
                x      = matrix.e;
                y      = matrix.f;
            }, () => {
                pane   = undefined;
                matrix = undefined;
            });
        }

        /**
         * On mouse over node, bring it to the front
         */
        this.domEvents.on("mouseover", ".node", (ev, target, root) => {
            if (this.workflow.querySelector(".edge.dragged")) {
                return;
            }
            target!.parentElement!.appendChild(target!);
        });

    }

    private clearCanvas() {
        this.domEvents.detachAll();
        this.workflow.innerHTML = "";
        this.workflow.setAttribute("transform", "matrix(1,0,0,1,0,0)");
        this.workflow.setAttribute("class", "workflow");
    }

    private hookPlugins() {

        this.plugins.forEach(plugin => {

            plugin.registerOnBeforeChange!(event => {
                this.eventHub.emit("beforeChange", event);
            });

            plugin.registerOnAfterChange!(event => {
                this.eventHub.emit("afterChange", event);
            });

            plugin.registerOnAfterRender!(event => {
                this.eventHub.emit("afterRender", event);
            });
        });
    }

    private invokePlugins(methodName: keyof SVGPlugin, ...args: any[]) {
        this.plugins.forEach(plugin => {
            if (typeof plugin[methodName] === "function") {
                (plugin[methodName] as Function)(...args);
            }
        });
    }

    /**
     * Listener for “connection.create” event on model that renders new edges on canvas
     */
    private onConnectionCreate(source: Connectable, destination: Connectable): void {

        if (!source.isVisible || !destination.isVisible) {
            return;
        }

        const sourceID      = source.connectionId;
        const destinationID = destination.connectionId;

        GraphEdge.spawnBetweenConnectionIDs(this.workflow, sourceID, destinationID);
    }

    /**
     * Listener for "connection.remove" event on the model that disconnects nodes
     */
    private onConnectionRemove(source: Connectable, destination: Connectable): void {
        if (!source.isVisible || !destination.isVisible) {
            return;
        }

        const sourceID      = source.connectionId;
        const destinationID = destination.connectionId;

        const edge = this.svgRoot.querySelector(`.edge[data-source-connection="${sourceID}"][data-destination-connection="${destinationID}"`);
        edge!.remove();
    }

    /**
     * Listener for “input.create” event on model that renders workflow inputs
     */
    private onInputCreate(input: WorkflowInputParameterModel): void {
        if (!input.isVisible) {
            return;
        }

        const patched       = GraphNode.patchModelPorts(input);
        const graphTemplate = GraphNode.makeTemplate(patched, this.labelScale);

        const el = TemplateParser.parse(graphTemplate)!;
        this.workflow.appendChild(el);

    }

    /**
     * Listener for “output.create” event on model that renders workflow outputs
     */
    private onOutputCreate(output: WorkflowOutputParameterModel): void {

        if (!output.isVisible) {
            return;
        }

        const patched       = GraphNode.patchModelPorts(output);
        const graphTemplate = GraphNode.makeTemplate(patched, this.labelScale);

        const el = TemplateParser.parse(graphTemplate)!;
        this.workflow.appendChild(el);
    }

    private onStepCreate(step: StepModel) {
        // if the step doesn't have x & y coordinates, check if they are in the run property
        if (!step.customProps["sbg:x"] && step.run.customProps && step.run.customProps["sbg:x"]) {

            Object.assign(step.customProps, {
                "sbg:x": step.run.customProps["sbg:x"],
                "sbg:y": step.run.customProps["sbg:y"]
            });

            // remove them from the run property once finished
            delete step.run.customProps["sbg:x"];
            delete step.run.customProps["sbg:y"];
        }

        const template = GraphNode.makeTemplate(step, this.labelScale);
        const element  = TemplateParser.parse(template)!;
        this.workflow.appendChild(element);
    }


    private onStepChange(change: StepModel) {
        const title = this.workflow.querySelector(`.step[data-id="${change.connectionId}"] .title`) as SVGTextElement;
        if (title) {
            title.textContent = change.label;
        }
    }

    private onInputPortShow(input: WorkflowStepInputModel) {

        const stepEl = this.svgRoot.querySelector(`.step[data-connection-id="${input.parentStep.connectionId}"]`) as SVGElement;
        new StepNode(stepEl, input.parentStep).update();
    }

    private onInputPortHide(input: WorkflowStepInputModel) {
        const stepEl = this.svgRoot.querySelector(`.step[data-connection-id="${input.parentStep.connectionId}"]`) as SVGElement;
        new StepNode(stepEl, input.parentStep).update();
    }

    private onOutputPortCreate(output: WorkflowStepOutputModel) {
        const stepEl = this.svgRoot.querySelector(`.step[data-connection-id="${output.parentStep.connectionId}"]`) as SVGElement;
        new StepNode(stepEl, output.parentStep).update();
    }

    private onOutputPortRemove(output: WorkflowStepOutputModel) {
        const stepEl = this.svgRoot.querySelector(`.step[data-connection-id="${output.parentStep.connectionId}"]`) as SVGElement;
        new StepNode(stepEl, output.parentStep).update();
    }

    /**
     * Listener for "step.remove" event on model which removes steps
     */
    private onStepRemove(step: StepModel) {
        const stepEl = this.svgRoot.querySelector(`.step[data-connection-id="${step.connectionId}"]`) as SVGElement;
        stepEl.remove();
    }

    /**
     * Listener for "input.remove" event on model which removes inputs
     */
    private onInputRemove(input: WorkflowInputParameterModel) {
        if (!input.isVisible) {
            return;
        }
        const inputEl = this.svgRoot.querySelector(`.node.input[data-connection-id="${input.connectionId}"]`);
        inputEl!.remove();
    }

    /**
     * Listener for "output.remove" event on model which removes outputs
     */
    private onOutputRemove(output: WorkflowOutputParameterModel) {
        if (!output.isVisible) {
            return;
        }
        const outputEl = this.svgRoot.querySelector(`.node.output[data-connection-id="${output.connectionId}"]`);
        outputEl!.remove();
    }

    private makeID(length = 6) {
        let output    = "";
        const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz";

        for (let i = 0; i < length; i++) {
            output += charset.charAt(Math.floor(Math.random() * charset.length));
        }

        return output;
    }
}
