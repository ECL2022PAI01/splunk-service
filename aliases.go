package main

import "github.com/ECL2022PAI01/splunk-service/handler"

const UnhandleKeptnCloudEvent = "Unhandled Keptn Cloud Event : "

// ServiceName specifies the current services name (e.g., used as source when sending CloudEvents)
const ServiceName = "splunk-service"

// These variables facilitate tests
var cloudEventListener = CloudEventListener
var processKeptnCloudEvent = ProcessKeptnCloudEvent
var handleConfigureMonitoringTriggeredEvent = handler.HandleConfigureMonitoringTriggeredEvent
var handleGetSliTriggeredEvent = handler.HandleGetSliTriggeredEvent
