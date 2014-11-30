App.Node = App.ArvadosModel.extend {
    hostname: DS.attr 'string'
    domain: DS.attr 'string'
    crunchWorkerState: DS.attr 'string'
    ipAddress: DS.attr 'string'
    firstPingAt: DS.attr 'date'
    lastPingAt: DS.attr 'date'
    slot_number: DS.attr 'number'
    job_uuid: DS.attr 'string'
    status: DS.attr 'string'
    properties: DS.attr 'string'
    info: DS.attr 'string'
    nameservers: DS.attr 'string'
}
