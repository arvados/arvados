import {Workflow} from "../..";
import {PluginBase} from "../plugin-base";

export class SelectionPlugin extends PluginBase {

    static edgePortsDelimiter = "$!$";
    private svg: SVGSVGElement;
    private selection = new Map<string, "edge" | "node">();
    private cleanups: Function[] = [];
    private detachModelEvents: Function | undefined;

    private selectionChangeCallbacks: Function[] = [];

    private css = {
        selected: "__selection-plugin-selected",
        highlight: "__selection-plugin-highlight",
        fade: "__selection-plugin-fade",
        plugin: "__plugin-selection"
    };

    registerWorkflow(workflow: Workflow): void {
        super.registerWorkflow(workflow);

        this.svg = this.workflow.svgRoot;

        this.svg.classList.add(this.css.plugin);

        const clickListener = this.onClick.bind(this);
        this.svg.addEventListener("click", clickListener);
        this.cleanups.push(() => this.svg.removeEventListener("click", clickListener));
    }

    afterRender() {
        this.restoreSelection();
    }

    afterModelChange(): void {
        if (typeof this.detachModelEvents === "function") {
            this.detachModelEvents();
        }

        this.detachModelEvents = this.bindModelEvents();
    }

    destroy() {

        if (this.detachModelEvents) {
            this.detachModelEvents();
        }
        this.detachModelEvents = undefined;

        this.svg.classList.remove(this.css.plugin);

        for (const fn of this.cleanups) {
            fn();
        }
    }

    clearSelection(): void {

        const selection: any  = this.svg.querySelectorAll(`.${this.css.selected}`);
        const highlights: any = this.svg.querySelectorAll(`.${this.css.highlight}`);

        for (const el of selection) {
            el.classList.remove(this.css.selected);
        }

        for (const el of highlights) {
            el.classList.remove(this.css.highlight);
        }

        this.svg.classList.remove(this.css.fade);

        this.selection.clear();

        this.emitChange(null);
    }

    getSelection() {
        return this.selection;
    }

    registerOnSelectionChange(fn: (node: any) => any) {
        this.selectionChangeCallbacks.push(fn);
    }

    selectStep(stepID: string) {
        const query = `[data-connection-id="${stepID}"]`;
        const el = this.svg.querySelector(query) as SVGElement;

        if (el) {
            this.materializeClickOnElement(el);
        }

    }

    private bindModelEvents() {

        const handler = () => this.restoreSelection();
        const cleanup: any[] = [];
        const events  = ["connection.create", "connection.remove"];

        for (const ev of events) {
            const dispose = this.workflow.model.on(ev as any, handler);
            cleanup.push(() => dispose.dispose());
        }

        return () => cleanup.forEach(fn => fn());
    }

    private restoreSelection() {
        this.selection.forEach((type, connectionID) => {

            if (type === "node") {

                const el = this.svg.querySelector(`[data-connection-id="${connectionID}"]`) as SVGElement;

                if (el) {
                    this.selectNode(el);
                }

            } else if (type === "edge") {

                const [sID, dID]   = connectionID.split(SelectionPlugin.edgePortsDelimiter);
                const edgeSelector = `[data-source-connection="${sID}"][data-destination-connection="${dID}"]`;

                const edge = this.svg.querySelector(edgeSelector) as SVGElement;

                if (edge) {
                    this.selectEdge(edge);
                }

            }
        });
    }

    private onClick(click: MouseEvent): void {
        const target = click.target as SVGElement;

        this.clearSelection();

        this.materializeClickOnElement(target);
    }

    private materializeClickOnElement(target: SVGElement) {

        let element: SVGElement | undefined;

        if ((element = this.workflow.findParent(target, "node"))) {
            this.selectNode(element);
            this.selection.set(element.getAttribute("data-connection-id")!, "node");
            this.emitChange(element);

        } else if ((element = this.workflow.findParent(target, "edge"))) {
            this.selectEdge(element);
            const cid = [
                element.getAttribute("data-source-connection"),
                SelectionPlugin.edgePortsDelimiter,
                element.getAttribute("data-destination-connection")
            ].join("");

            this.selection.set(cid, "edge");
            this.emitChange(cid);
        }
    }

    private selectNode(element: SVGElement): void {
        // Fade everything on canvas so we can highlight only selected stuff
        this.svg.classList.add(this.css.fade);

        // Mark this node as selected
        element.classList.add(this.css.selected);
        // Highlight it in case there are no edges on the graph
        element.classList.add(this.css.highlight);

        // Take all adjacent edges since we should highlight them and move them above the other edges
        const nodeID        = element.getAttribute("data-id");
        const adjacentEdges: any = this.svg.querySelectorAll(
            `.edge[data-source-node="${nodeID}"],` +
            `.edge[data-destination-node="${nodeID}"]`
        );

        // Find the first node to be an anchor, so we can put all those edges just before that one.
        const firstNode = this.svg.getElementsByClassName("node")[0];

        for (const edge of adjacentEdges) {

            // Highlight each adjacent edge
            edge.classList.add(this.css.highlight);

            // Move it above other edges
            this.workflow.workflow.insertBefore(edge, firstNode);

            // Find all adjacent nodes so we can highlight them
            const sourceNodeID      = edge.getAttribute("data-source-node");
            const destinationNodeID = edge.getAttribute("data-destination-node");
            const connectedNodes: any    = this.svg.querySelectorAll(
                `.node[data-id="${sourceNodeID}"],` +
                `.node[data-id="${destinationNodeID}"]`
            );

            // Highlight each adjacent node
            for (const n of connectedNodes) {
                n.classList.add(this.css.highlight);
            }
        }
    }

    private selectEdge(element: SVGElement) {

        element.classList.add(this.css.highlight);
        element.classList.add(this.css.selected);

        const sourceNode = element.getAttribute("data-source-node");
        const destNode   = element.getAttribute("data-destination-node");
        const sourcePort = element.getAttribute("data-source-port");
        const destPort   = element.getAttribute("data-destination-port");

        const inputPortSelector  = `.node[data-id="${destNode}"] .input-port[data-port-id="${destPort}"]`;
        const outputPortSelector = `.node[data-id="${sourceNode}"] .output-port[data-port-id="${sourcePort}"]`;

        const connectedPorts: any = this.svg.querySelectorAll(`${inputPortSelector}, ${outputPortSelector}`);

        for (const port of connectedPorts) {
            port.classList.add(this.css.highlight);
        }
    }

    private emitChange(change: any) {
        for (const fn of this.selectionChangeCallbacks) {
            fn(change);
        }
    }
}
