import {PluginBase} from "../plugin-base";
import {SelectionPlugin} from "../selection/selection";
import {StepModel, WorkflowInputParameterModel, WorkflowOutputParameterModel} from "cwlts/models";

export class DeletionPlugin extends PluginBase {

    private boundDeleteFunction = this.onDelete.bind(this);

    afterRender(): void {
        this.attachDeleteBehavior();
    }

    onEditableStateChange(enable: boolean) {
        if (enable) {
            this.attachDeleteBehavior();
        } else {
            this.detachDeleteBehavior();
        }
    }

    private attachDeleteBehavior() {

        this.detachDeleteBehavior();
        window.addEventListener("keyup", this.boundDeleteFunction, true);
    }

    private detachDeleteBehavior() {
        window.removeEventListener("keyup", this.boundDeleteFunction, true);
    }

    private onDelete(ev: KeyboardEvent) {
        if ((ev.which !== 8 && ev.which !== 46) || !(ev.target instanceof SVGElement)) {
            return;
        }

        this.deleteSelection();
    }

    public deleteSelection() {
        const selection = this.workflow.getPlugin(SelectionPlugin);

        if (!selection || !this.workflow.editingEnabled) {
            return;
        }

        const selected = selection.getSelection();
        selected.forEach((type, id) => {
            if (type === "node") {
                const model = this.workflow.model.findById(id);

                if (model instanceof StepModel) {
                    this.workflow.model.removeStep(model);
                    selection.clearSelection();

                } else if (model instanceof WorkflowInputParameterModel) {
                    this.workflow.model.removeInput(model);
                    selection.clearSelection();

                } else if (model instanceof WorkflowOutputParameterModel) {

                    this.workflow.model.removeOutput(model);
                    selection.clearSelection();
                }
            } else {
                const [source, destination] = id.split(SelectionPlugin.edgePortsDelimiter);
                this.workflow.model.disconnect(source, destination);
                selection.clearSelection();
            }
        });
    }

    destroy() {
        this.detachDeleteBehavior();
    }
}