export class EventHub {
    public readonly handlers: { [event: string]: Function[] };

    constructor(validEventList: string[]) {
        this.handlers = validEventList.reduce((acc, ev) => Object.assign(acc, {[ev]: []}), {});
    }

    on(event: string, handler: Function) {
        this.guard(event, "subscribe to");
        this.handlers[event].push(handler);

        return () => this.off(event, handler);
    }

    off(event: string, handler: Function) {
        this.guard(event, "unsubscribe from");
        return this.handlers[event].splice(this.handlers[event].findIndex(h => handler === h), 1);
    }

    emit(event: string, ...data: any[]) {
        this.guard(event, "emit");
        for (let i = 0; i < this.handlers[event].length; i++) {
            this.handlers[event][i](...data);
        }
    }

    empty() {
        for (let event in this.handlers) {
            this.handlers[event] = [];
        }
    }

    private guard(event: string, verb: string) {
        if (!this.handlers[event]) {
            console.warn(`Trying to ${verb} a non-supported event “${event}”. 
            Supported events are: ${Object.keys(this.handlers).join(", ")}”`);
        }
    }
}
