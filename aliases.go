package main

const UnhanKeptnCE = "Unhandled Keptn Cloud Event : "

// ServiceName specifies the current services name (e.g., used as source when sending CloudEvents)
const ServiceName = "splunk-service"

// These variables facilitate tests
var handleConfMonitEvent = HandleConfigureMonitoringTriggeredEvent
var handleGetSli= HandleGetSliTriggeredEvent
var call_main = _main
var procKeptnCE = processKeptnCloudEvent