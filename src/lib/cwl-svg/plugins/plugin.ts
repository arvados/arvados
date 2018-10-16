import {Workflow} from '../graph/workflow';

export interface GraphChange {
    type: string;

}

export interface SVGPlugin {

    registerWorkflow?(workflow: Workflow): void;

    registerOnBeforeChange?(fn: (change: GraphChange) => void): void;

    registerOnAfterChange?(fn: (change: GraphChange) => void): void;

    registerOnAfterRender?(fn: (change: GraphChange) => void): void;

    afterRender?(): void;

    /**
     * Invoked when the underlying model instance changes.
     * Implementation should dispose listeners from the old model and attach listeners to the new one.
     */
    afterModelChange?(): void;

    onEditableStateChange?(enabled: boolean): void;

    /**
     * Invoked when a graph should be destroyed.
     * Implementations should remove attached DOM and model event listeners, as well as other stuff that
     * might be left in memory.
     */
    destroy?(): void;
}